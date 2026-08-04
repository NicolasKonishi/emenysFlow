(() => {
  "use strict";

  const DB_NAME = "buffetflow-offline";
  const DB_VERSION = 1;
  const DEVICE_KEY = "buffetflow_device_id";
  let registration;
  let synchronizing = false;

  const uuid = () => self.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  const deviceID = () => {
    let value = localStorage.getItem(DEVICE_KEY);
    if (!value) {
      value = uuid();
      localStorage.setItem(DEVICE_KEY, value);
    }
    return value;
  };

  function openDatabase() {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, DB_VERSION);
      request.onupgradeneeded = () => {
        const db = request.result;
        if (!db.objectStoreNames.contains("meta")) db.createObjectStore("meta", { keyPath: "key" });
        if (!db.objectStoreNames.contains("operations")) {
          const store = db.createObjectStore("operations", { keyPath: "client_operation_id" });
          store.createIndex("status", "status");
        }
        if (!db.objectStoreNames.contains("photos")) {
          const store = db.createObjectStore("photos", { keyPath: "client_upload_id" });
          store.createIndex("status", "status");
        }
        if (!db.objectStoreNames.contains("conflicts")) db.createObjectStore("conflicts", { keyPath: "client_operation_id" });
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
  }

  async function useStore(name, mode, callback) {
    const db = await openDatabase();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction(name, mode);
      const store = transaction.objectStore(name);
      let result;
      try { result = callback(store); } catch (error) { reject(error); return; }
      transaction.oncomplete = () => resolve(result?.result ?? result);
      transaction.onerror = () => reject(transaction.error);
    }).finally(() => db.close());
  }

  const getRecord = (store, key) => useStore(store, "readonly", (target) => target.get(key));
  const putRecord = (store, value) => useStore(store, "readwrite", (target) => target.put(value));
  const deleteRecord = (store, key) => useStore(store, "readwrite", (target) => target.delete(key));
  const getAll = (store) => useStore(store, "readonly", (target) => target.getAll());

  async function updateStatus(message, state) {
    const resolvedState = state || (navigator.onLine ? "online" : "offline");
    const connection = document.querySelector("[data-connection-state]");
    if (connection) {
      connection.textContent = message || (navigator.onLine ? "Online" : "Offline");
      connection.dataset.state = resolvedState;
    }
    const operations = await getAll("operations").catch(() => []);
    const photos = await getAll("photos").catch(() => []);
    const pending = operations.filter((item) => ["pending", "syncing", "failed"].includes(item.status)).length
      + photos.filter((item) => ["pending", "syncing", "failed"].includes(item.status)).length;
    const statusBar = document.querySelector("[data-sync-status-bar]");
    if (statusBar) statusBar.hidden = !(resolvedState === "syncing" || resolvedState === "error" || pending > 0);
  }

  async function refreshBootstrap() {
    if (!navigator.onLine || !document.querySelector("[data-sync-status-bar]")) return;
    const response = await fetch("/api/offline/bootstrap", { credentials: "same-origin", headers: { Accept: "application/json", "X-BuffetFlow-Client": "pwa" } });
    if (!response.ok) throw new Error("Não foi possível atualizar os dados offline.");
    const data = await response.json();
    await putRecord("meta", { key: "bootstrap", value: data, updated_at: new Date().toISOString() });
    await putRecord("meta", { key: "last_sync", value: data.synced_at || new Date().toISOString() });
  }

  function formPayload(form) {
    const payload = {};
    new FormData(form).forEach((value, key) => {
      if (value instanceof File) return;
      if (Object.prototype.hasOwnProperty.call(payload, key)) {
        if (!Array.isArray(payload[key])) payload[key] = [payload[key]];
        payload[key].push(value);
      } else payload[key] = value;
    });
    form.querySelectorAll('input[type="checkbox"]').forEach((input) => { payload[input.name] = input.checked; });
    const eventRoot = form.closest("[data-offline-event-id]");
    const match = form.action.match(/\/events\/(\d+)/);
    payload.event_id = Number(eventRoot?.dataset.offlineEventId || match?.[1] || form.dataset.entityId || 0);
    return payload;
  }

  async function queueForm(form) {
    const entityRoot = form.closest("[data-entity-id]");
    const entityMatch = form.action.match(/\/(?:items|shortages)\/(\d+)(?:\/|$)/);
    const operation = {
      client_operation_id: uuid(),
      device_id: deviceID(),
      operation_type: form.dataset.operationType,
      entity_type: form.dataset.entityType,
      entity_id: Number(form.dataset.entityId || entityRoot?.dataset.entityId || entityMatch?.[1] || 0),
      base_version: Number(form.dataset.version || form.querySelector('[name="version"]')?.value || entityRoot?.dataset.version || 0),
      payload: formPayload(form),
      local_date: new Date().toISOString(),
      attempts: 0,
      last_attempt: null,
      last_error: "",
      status: "pending"
    };
    await putRecord("operations", operation);
    form.dataset.offlineQueued = "true";
    const button = form.querySelector('button[type="submit"], button:not([type])');
    if (button) {
      button.dataset.originalLabel ||= button.textContent;
      button.textContent = "Salvo no aparelho";
    }
    await updateStatus("Offline — alteração salva no aparelho", "pending");
    return operation;
  }

  async function queueLoadingDecision(eventID, itemID, decision, missingQuantity) {
    const operation = {
      client_operation_id: uuid(), device_id: deviceID(), operation_type: "mobile_loading_decision", entity_type: "checklist_item",
      entity_id: itemID, base_version: 0, payload: { event_id: eventID, decision, missing_quantity: missingQuantity },
      local_date: new Date().toISOString(), attempts: 0, last_attempt: null, last_error: "", status: "pending"
    };
    await putRecord("operations", operation);
    await updateStatus("Offline — carregamento salvo no aparelho", "pending");
  }

  function initializeOperationalQuickLoading(root = document) {
    if (!matchMedia("(max-width: 820px)").matches) return;
    root.querySelectorAll('.operational-hub form[data-operation-type="update_quantity"] input[name="stage"][value="loading"]').forEach((stage) => {
      const form = stage.closest("form");
      if (!form || form.dataset.quickLoadingInitialized === "true") return;
      form.dataset.quickLoadingInitialized = "true";
      form.classList.add("mobile-detail-form");
      const match = form.action.match(/\/events\/(\d+)\/operation\/items\/(\d+)\/quantity/);
      const quantity = form.querySelector('input[name="quantity"]');
      if (!match || !quantity) return;
      const eventID = Number(match[1]);
      const itemID = Number(match[2]);
      const required = Number(quantity.max || quantity.value || 0);
      const actions = document.createElement("div");
      actions.className = "mobile-quick-actions";
      actions.innerHTML = '<button type="button" class="button primary" data-quick-complete>✓ Concluído</button><button type="button" class="button danger" data-quick-missing>Falta quantidade</button><div class="mobile-missing-editor" hidden><label>Quanto falta?<input type="number" min="0.01" step="0.01" required></label><button type="button" class="button danger" data-quick-missing-save>Confirmar falta</button></div><small aria-live="polite"></small>';
      form.before(actions);
      const status = actions.querySelector("small");
      const editor = actions.querySelector(".mobile-missing-editor");
      const missingInput = editor.querySelector("input");
      const submitDecision = async (decision, missing) => {
        actions.querySelectorAll("button,input").forEach((control) => { control.disabled = true; });
        status.textContent = navigator.onLine ? "Salvando…" : "Salvando no aparelho…";
        try {
          if (!navigator.onLine) {
            await queueLoadingDecision(eventID, itemID, decision, missing);
            status.textContent = decision === "complete" ? "Concluído — sincronização pendente" : `Falta ${missing} — sincronização pendente`;
            return;
          }
          const body = new URLSearchParams({ decision, missing_quantity: String(missing || 0) });
          const response = await fetch(`/events/${eventID}/operation/loading/items/${itemID}`, { method: "POST", body, credentials: "same-origin", headers: { Accept: "application/json", "X-BuffetFlow-Client": "pwa" } });
          const result = await response.json();
          if (!response.ok) throw new Error(result.error || "Não foi possível salvar.");
          location.reload();
        } catch (error) {
          status.textContent = error.message || "Não foi possível salvar.";
        } finally {
          actions.querySelectorAll("button,input").forEach((control) => { control.disabled = false; });
        }
      };
      actions.querySelector("[data-quick-complete]").addEventListener("click", () => submitDecision("complete", 0));
      actions.querySelector("[data-quick-missing]").addEventListener("click", () => { editor.hidden = !editor.hidden; if (!editor.hidden) missingInput.focus(); });
      actions.querySelector("[data-quick-missing-save]").addEventListener("click", () => {
        const missing = Number(missingInput.value);
        if (!Number.isFinite(missing) || missing <= 0 || missing > required) {
          status.textContent = `Informe uma falta entre 0 e ${required}.`;
          return;
        }
        submitDecision("missing", missing);
      });
    });
  }

  async function queuePhotos(form) {
    const files = Array.from(form.querySelector('input[type="file"]')?.files || []);
    if (!files.length) throw new Error("Selecione ao menos uma foto.");
    for (const file of files) {
      if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type) || file.size > 8 * 1024 * 1024) {
        throw new Error("Use fotos JPG, PNG ou WEBP de até 8 MB.");
      }
      const fields = formPayload(form);
      delete fields.photos;
      const record = { client_upload_id: uuid(), endpoint: form.action, file, fields, filename: file.name, mime_type: file.type, status: "pending", attempts: 0, last_error: "", created_at: new Date().toISOString() };
      await putRecord("photos", record);
      const preview = document.createElement("figure");
      preview.className = "offline-photo-preview";
      const image = document.createElement("img");
      image.src = URL.createObjectURL(file);
      image.alt = fields.caption || "Foto aguardando envio";
      const caption = document.createElement("figcaption");
      caption.textContent = `${file.name} · upload pendente`;
      preview.append(image, caption);
      form.before(preview);
    }
    form.reset();
    await updateStatus("Fotos salvas no aparelho", "pending");
  }

  async function syncPhotos() {
    const photos = (await getAll("photos")).filter((photo) => photo.status === "pending" || photo.status === "failed");
    for (const photo of photos) {
      photo.status = "syncing";
      photo.attempts += 1;
      photo.last_attempt = new Date().toISOString();
      await putRecord("photos", photo);
      const body = new FormData();
      Object.entries(photo.fields || {}).forEach(([key, value]) => body.append(key, String(value)));
      body.append("client_upload_id", photo.client_upload_id);
      body.append("photos", photo.file, photo.filename);
      try {
        const response = await fetch(photo.endpoint, { method: "POST", body, credentials: "same-origin", headers: { Accept: "application/json", "X-BuffetFlow-Client": "pwa" } });
        if (!response.ok) throw new Error((await response.json().catch(() => ({}))).error || `HTTP ${response.status}`);
        await deleteRecord("photos", photo.client_upload_id);
      } catch (error) {
        photo.status = "failed";
        photo.last_error = error.message || "Falha no upload";
        await putRecord("photos", photo);
      }
    }
  }

  async function syncOperations() {
    if (!navigator.onLine || synchronizing) return;
    const all = await getAll("operations").catch(() => []);
    const photos = await getAll("photos").catch(() => []);
    const hasPendingUpdates = all.some((item) => item.status === "pending" || item.status === "failed")
      || photos.some((item) => item.status === "pending" || item.status === "failed");
    if (!hasPendingUpdates) {
      refreshBootstrap().catch(() => null);
      await updateStatus();
      return;
    }
    synchronizing = true;
    await updateStatus("Sincronizando…", "syncing");
    try {
      const pending = all.filter((item) => item.status === "pending" || item.status === "failed").slice(0, 100);
      if (pending.length) {
        for (const item of pending) {
          item.status = "syncing";
          item.attempts += 1;
          item.last_attempt = new Date().toISOString();
          await putRecord("operations", item);
        }
        const response = await fetch("/api/sync/operations", { method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json", Accept: "application/json", "X-BuffetFlow-Client": "pwa" }, body: JSON.stringify(pending) });
        if (!response.ok) throw new Error(`Sincronização recusada (HTTP ${response.status}).`);
        const body = await response.json();
        for (const result of body.results || []) {
          const operation = pending.find((item) => item.client_operation_id === result.client_operation_id);
          if (!operation) continue;
          operation.status = result.status;
          operation.last_error = result.error || "";
          operation.server_version = result.version || 0;
          if (result.entity_id) operation.entity_id = result.entity_id;
          await putRecord("operations", operation);
          if (result.status === "conflict") await putRecord("conflicts", { ...operation, server_snapshot: result.server_snapshot, server_version: result.version, detected_at: new Date().toISOString() });
        }
      }
      await syncPhotos();
      await refreshBootstrap();
      renderConflicts();
      await updateStatus();
    } catch (error) {
      const operations = await getAll("operations").catch(() => []);
      for (const operation of operations.filter((item) => item.status === "syncing")) {
        operation.status = "failed";
        operation.last_error = error.message || "Falha de sincronização";
        await putRecord("operations", operation);
      }
      await updateStatus(navigator.onLine ? "Erro de sincronização" : "Offline", navigator.onLine ? "error" : "offline");
    } finally {
      synchronizing = false;
    }
  }

  async function renderConflicts() {
    const conflicts = await getAll("conflicts").catch(() => []);
    const panel = document.querySelector("[data-conflict-panel]");
    const list = document.querySelector("[data-conflict-list]");
    if (!panel || !list) return;
    panel.hidden = conflicts.length === 0;
    list.replaceChildren();
    conflicts.forEach((conflict) => {
      const card = document.createElement("article");
      card.innerHTML = `<strong>${conflict.entity_type}</strong><p>O servidor foi atualizado depois da sua cópia local.</p><details><summary>Comparar versões</summary><div class="conflict-versions"><pre></pre><pre></pre></div></details><div class="conflict-actions"><button class="button secondary small" data-choice="server">Manter servidor</button><button class="button secondary small" data-choice="merge">Mesclar campos</button><button class="button primary small" data-choice="local">Manter local</button></div>`;
      const versions = card.querySelectorAll("pre");
      versions[0].textContent = `Local\n${JSON.stringify(conflict.payload, null, 2)}`;
      versions[1].textContent = `Servidor\n${JSON.stringify(conflict.server_snapshot, null, 2)}`;
      card.querySelectorAll("[data-choice]").forEach((button) => button.addEventListener("click", async () => {
        if (button.dataset.choice === "server") {
          conflict.status = "synced";
        } else {
          conflict.base_version = conflict.server_version || conflict.base_version;
          conflict.status = "pending";
          conflict.last_error = "";
          if (button.dataset.choice === "merge" && conflict.server_snapshot && conflict.operation_type === "update_event_draft") {
            conflict.payload = { ...conflict.server_snapshot, ...conflict.payload };
          }
        }
        const clean = { ...conflict };
        delete clean.server_snapshot;
        delete clean.server_version;
        delete clean.detected_at;
        await putRecord("operations", clean);
        await deleteRecord("conflicts", conflict.client_operation_id);
        await renderConflicts();
        await syncOperations();
      }));
      list.append(card);
    });
  }

  async function renderOfflineHome() {
    const root = document.querySelector("[data-offline-home]");
    if (!root) return;
    const saved = await getRecord("meta", "bootstrap").catch(() => null);
    if (!saved?.value) {
      root.innerHTML = "<p>Nenhum evento foi sincronizado neste aparelho. Conecte-se e abra o emenysFlow ao menos uma vez.</p>";
      return;
    }
    const expiration = new Date(saved.value.offline_access_expires_at || 0);
    if (expiration < new Date()) {
      root.innerHTML = "<p>O acesso offline expirou. Conecte-se novamente para validar sua sessão.</p>";
      return;
    }
    root.replaceChildren();
    (saved.value.events || []).forEach((bundle) => {
      const card = document.createElement("details");
      card.className = "offline-event-card";
      const event = bundle.event;
      const items = bundle.checklist?.items || bundle.checklist?.Items || [];
      const menuSections = bundle.menu || [];
      card.innerHTML = `<summary><strong>${event.name || event.Name}</strong><span>${event.client_name || event.ClientName} · ${event.guest_count || event.GuestCount} pessoas</span></summary><div class="offline-event-detail"><h3>Cardápio</h3><div data-menu></div><h3>Serviços</h3><p>${(bundle.service_ids || []).length} modelo(s) de serviço vinculado(s)</p><h3>Checklist</h3><ul data-checklist></ul></div>`;
      const menu = card.querySelector("[data-menu]");
      menu.textContent = menuSections.map((section) => `${section.name || section.Name || section.section_name}: ${(section.items || section.Items || []).filter((item) => !(item.was_removed || item.WasRemoved)).map((item) => item.display_name || item.DisplayName || item.name || item.Name).join(", ")}`).join(" · ") || "Sem cardápio sincronizado.";
      const checklist = card.querySelector("[data-checklist]");
      items.forEach((item) => {
        const row = document.createElement("li");
        row.textContent = `${item.name || item.Name}: ${item.required_quantity ?? item.RequiredQuantity} ${item.unit || item.Unit}`;
        checklist.append(row);
      });
      root.append(card);
    });
  }

  async function clearOfflineData() {
    await new Promise((resolve) => {
      const request = indexedDB.deleteDatabase(DB_NAME);
      request.onsuccess = request.onerror = request.onblocked = () => resolve();
    });
    const keys = await caches.keys();
    await Promise.all(keys.filter((key) => key.startsWith("buffetflow-")).map((key) => caches.delete(key)));
    localStorage.removeItem(DEVICE_KEY);
  }

  document.addEventListener("submit", async (event) => {
    const logout = event.target.closest('form[action="/logout"]');
    if (logout) {
      event.preventDefault();
      await clearOfflineData();
      await fetch("/logout", { method: "POST", credentials: "same-origin" }).catch(() => null);
      location.href = "/login";
      return;
    }
    const photoForm = event.target.closest("form[data-offline-photo-form]");
    if (photoForm && !navigator.onLine) {
      event.preventDefault();
      try { await queuePhotos(photoForm); } catch (error) { window.alert(error.message); }
      return;
    }
    const form = event.target.closest("form[data-offline-form]");
    if (!form || navigator.onLine || !form.dataset.operationType) return;
    event.preventDefault();
    try { await queueForm(form); } catch (_error) { window.alert("Não foi possível salvar a alteração neste aparelho."); }
  }, true);

  document.addEventListener("click", (event) => {
    if (event.target.closest("[data-sync-now]")) syncOperations();
    if (event.target.closest("[data-close-conflicts]")) document.querySelector("[data-conflict-panel]").hidden = true;
  });
  window.addEventListener("online", () => syncOperations());
  window.addEventListener("offline", () => updateStatus("Offline — alterações serão guardadas", "offline"));
  navigator.serviceWorker?.addEventListener("message", (event) => {
    if (event.data?.type === "SYNC_REQUESTED") syncOperations();
  });
  let reloadingForUpdate = false;
  navigator.serviceWorker?.addEventListener("controllerchange", () => {
    if (reloadingForUpdate) return;
    reloadingForUpdate = true;
    location.reload();
  });

  document.addEventListener("DOMContentLoaded", async () => {
    if ("serviceWorker" in navigator) {
      registration = await navigator.serviceWorker.register("/sw.js").catch(() => null);
      registration?.waiting?.postMessage({ type: "SKIP_WAITING" });
      registration?.update().catch(() => null);
    }
    await updateStatus();
    await renderConflicts();
    await renderOfflineHome();
    initializeOperationalQuickLoading();
    if (navigator.onLine) {
      await syncOperations();
      refreshBootstrap().catch(() => null);
    }
  });
})();
