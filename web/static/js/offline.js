(() => {
  "use strict";

  const DB_NAME = "buffetflow-offline";
  const DB_VERSION = 2;
  const DEVICE_KEY = "buffetflow_device_id";
  const SYNC_PREF_KEY = "buffetflow_sync_enabled";
  let registration;
  let synchronizing = false;

  const isSyncEnabled = () => localStorage.getItem(SYNC_PREF_KEY) === "1";
  const setSyncEnabled = (enabled) => localStorage.setItem(SYNC_PREF_KEY, enabled ? "1" : "0");
  const canUseOfflineData = () => Boolean(document.querySelector(".app-shell, [data-offline-hub], [data-sync-status-bar], [data-offline-home], [data-download-offline]"));
  const HEALTH_URL = "/api/health";
  const WORKSPACE_KEY = "buffetflow_workspace";
  const WAS_OFFLINE_KEY = "buffetflow_was_offline";
  const PROBE_MS = 8000;
  let serviceReachable = null;
  let reconnectDismissed = false;
  let switchingToOffline = false;
  let probeTimer = 0;
  let failCount = 0;

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
        if (!db.objectStoreNames.contains("layout_drafts")) db.createObjectStore("layout_drafts", { keyPath: "key" });
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
    const pendingLabel = document.querySelector("[data-pending-count]");
    if (pendingLabel) pendingLabel.textContent = pending === 1 ? "1 operação pendente" : `${pending} operações pendentes`;
    const lastSync = document.querySelector("[data-last-sync]");
    if (lastSync) {
      const saved = await getRecord("meta", "last_sync").catch(() => null);
      lastSync.textContent = saved?.value ? `Último download: ${new Date(saved.value).toLocaleString("pt-BR")}` : "Ainda não salvo neste aparelho";
    }
    updateSyncPreferenceStatus(pending);
  }

  function updateSyncPreferenceStatus(pending = 0) {
    const status = document.querySelector("[data-sync-preference-status]");
    if (!status) return;
    if (!navigator.onLine) {
      status.textContent = "Sem conexão. As alterações ficam neste aparelho.";
      return;
    }
    if (isSyncEnabled()) {
      status.textContent = pending > 0
        ? "Sincronização ligada. As alterações pendentes sobem automaticamente."
        : "Sincronização ligada. Alterações da checklist e do layout sobem quando houver conexão.";
      return;
    }
    status.textContent = pending > 0
      ? "Sincronização desligada. Há alterações só neste aparelho — envie se quiser."
      : "Sincronização automática desligada. Use “Sincronizar agora” só se quiser enviar ao online.";
  }

  function currentWorkspace() {
    if (document.body.classList.contains("workspace-offline") || document.querySelector("[data-offline-home]")) return "offline";
    if (document.body.classList.contains("workspace-online")) return "online";
    return document.body.dataset.workspace || "";
  }

  function isOfflineCapablePath(path = location.pathname) {
    return path === "/offline"
      || path.startsWith("/layouts")
      || /\/events\/\d+\/(operation|layout)/.test(path)
      || path.endsWith("/offline.html");
  }

  function persistWorkspace(mode) {
    localStorage.setItem(WORKSPACE_KEY, mode);
    document.cookie = `buffet_workspace=${mode}; Path=/; Max-Age=${365 * 24 * 60 * 60}; SameSite=Lax`;
    document.body.classList.remove("workspace-online", "workspace-offline");
    document.body.classList.add(`workspace-${mode}`);
    document.body.dataset.workspace = mode;
    document.querySelectorAll(".workspace-label").forEach((node) => {
      node.textContent = mode === "offline" ? (node.closest(".mobile-header") ? "Offline" : "Modo offline") : (node.closest(".mobile-header") ? "emenysFlow" : "Modo online");
    });
  }

  function onOfflineHub() {
    const path = location.pathname;
    return path === "/offline" || path.endsWith("/offline.html") || Boolean(document.querySelector("[data-offline-home], [data-offline-hub]"));
  }

  function markDisconnected() {
    sessionStorage.setItem(WAS_OFFLINE_KEY, "1");
  }

  function wasDisconnected() {
    return sessionStorage.getItem(WAS_OFFLINE_KEY) === "1";
  }

  function clearDisconnected() {
    sessionStorage.removeItem(WAS_OFFLINE_KEY);
  }

  function openOnlineMode() {
    reconnectDismissed = true;
    clearDisconnected();
    persistWorkspace("online");
    showReconnectBanner(false);
    if (onOfflineHub()) location.assign("/");
  }

  function showReconnectBanner(visible) {
    const banner = document.querySelector("[data-reconnect-banner]");
    if (!banner) return;
    banner.hidden = !visible;
  }

  async function probeService() {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 4000);
    try {
      const response = await fetch(HEALTH_URL, {
        cache: "no-store",
        credentials: "same-origin",
        headers: { Accept: "application/json", "X-BuffetFlow-Client": "pwa" },
        signal: controller.signal
      });
      const data = await response.json().catch(() => ({}));
      return response.ok && data.ok === true;
    } catch (_error) {
      return false;
    } finally {
      clearTimeout(timer);
    }
  }

  async function applyServiceState(reachable) {
    const previous = serviceReachable;
    serviceReachable = reachable;
    if (!document.body.classList.contains("app-shell") && !document.body.classList.contains("offline-shell")) {
      return reachable;
    }
    if (!reachable) {
      failCount += 1;
      if (failCount < 2 && previous !== false && navigator.onLine) return reachable;
      reconnectDismissed = false;
      markDisconnected();
      showReconnectBanner(false);
      persistWorkspace("offline");
      await updateStatus("Sem conexão com o emenysFlow", "offline");
      if (!isOfflineCapablePath() && !switchingToOffline) {
        switchingToOffline = true;
        location.replace("/offline");
      }
      return reachable;
    }
    failCount = 0;
    if (wasDisconnected() && onOfflineHub() && !reconnectDismissed) {
      persistWorkspace("offline");
      showReconnectBanner(true);
      await updateStatus("Serviço online — continue offline ou abra o sistema completo", "online");
    } else {
      if (!onOfflineHub()) clearDisconnected();
      persistWorkspace("online");
      showReconnectBanner(false);
      await updateStatus("Conectado ao emenysFlow", "online");
      fetch("/offline", { credentials: "same-origin" }).catch(() => null);
    }
    if (previous === false && reachable && isSyncEnabled()) {
      syncOperations(true).catch(() => null);
    }
    return reachable;
  }

  async function watchServiceConnection() {
    const reachable = await probeService();
    await applyServiceState(reachable);
    if (reachable && canUseOfflineData()) refreshBootstrap().catch(() => null);
    return reachable;
  }

  async function refreshBootstrap() {
    if (!canUseOfflineData()) return;
    const reachable = serviceReachable === null ? await probeService() : serviceReachable;
    if (!reachable) return;
    const response = await fetch("/api/offline/bootstrap", { credentials: "same-origin", headers: { Accept: "application/json", "X-BuffetFlow-Client": "pwa" } });
    if (!response.ok) throw new Error("Não foi possível atualizar os dados offline.");
    const data = await response.json();
    await putRecord("meta", { key: "bootstrap", value: data, updated_at: new Date().toISOString() });
    await putRecord("meta", { key: "last_sync", value: data.synced_at || new Date().toISOString() });
    await updateStatus("Eventos salvos neste aparelho", "online");
    return data;
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
    if (form.hasAttribute("data-layout-form")) {
      form.dispatchEvent(new CustomEvent("layout-persist", { bubbles: false }));
    }
    const entityRoot = form.closest("[data-entity-id]");
    const entityMatch = form.action.match(/\/(?:items|shortages|layouts)\/(\d+)(?:\/|$)/);
    const payload = formPayload(form);
    if (form.dataset.layoutKey) payload.layout_key = form.dataset.layoutKey;
    if (form.hasAttribute("data-layout-form") && form.dataset.layoutKey) {
      const existing = await getAll("operations").catch(() => []);
      for (const item of existing) {
        if ((item.status === "pending" || item.status === "failed")
          && item.payload?.layout_key === form.dataset.layoutKey
          && ["save_event_layout", "save_standalone_layout"].includes(item.operation_type)) {
          await deleteRecord("operations", item.client_operation_id);
        }
      }
    }
    const operation = {
      client_operation_id: uuid(),
      device_id: deviceID(),
      operation_type: form.dataset.operationType,
      entity_type: form.dataset.entityType,
      entity_id: Number(form.dataset.entityId || entityRoot?.dataset.entityId || entityMatch?.[1] || 0),
      base_version: Number(form.dataset.version || form.querySelector('[name="version"]')?.value || entityRoot?.dataset.version || 0),
      payload,
      local_date: new Date().toISOString(),
      attempts: 0,
      last_attempt: null,
      last_error: "",
      status: "pending"
    };
    await putRecord("operations", operation);
    form.dataset.offlineQueued = "true";
    const button = form.querySelector('button[type="submit"], button:not([type])') || document.querySelector(`button[form="${form.id}"]`);
    if (button) {
      button.dataset.originalLabel ||= button.textContent;
      button.textContent = "Salvo no aparelho";
    }
    if (form.hasAttribute("data-layout-form") && form.dataset.layoutKey) {
      await deleteRecord("layout_drafts", form.dataset.layoutKey).catch(() => null);
      showLayoutOfflineNotice("Layout salvo no aparelho. Sincroniza ao voltar online.");
    }
    await updateStatus("Offline — alteração salva no aparelho", "pending");
    return operation;
  }

  function showLayoutOfflineNotice(message) {
    const editor = document.querySelector("[data-layout-editor]");
    if (!editor) return;
    let notice = editor.querySelector("[data-layout-offline-notice]");
    if (!notice) {
      notice = document.createElement("div");
      notice.className = "alert info layout-offline-notice";
      notice.dataset.layoutOfflineNotice = "1";
      notice.setAttribute("role", "status");
      editor.querySelector(".layout-planner-header")?.after(notice);
    }
    notice.textContent = message;
  }

  async function saveLayoutDraft(key, draft) {
    if (!key) return;
    await putRecord("layout_drafts", { key, ...draft, updated_at: new Date().toISOString() });
  }

  async function loadLayoutDraft(key) {
    if (!key) return null;
    return getRecord("layout_drafts", key).catch(() => null);
  }

  async function deleteLayoutDraft(key) {
    if (!key) return;
    await deleteRecord("layout_drafts", key).catch(() => null);
  }

  async function applyStandaloneLayoutSyncResult(operation, result) {
    if (operation.operation_type !== "save_standalone_layout" || result.status !== "synced" || !result.entity_id) return;
    const layoutKey = operation.payload?.layout_key;
    if (!layoutKey) return;
    const all = await getAll("operations").catch(() => []);
    for (const item of all) {
      if (item.client_operation_id === operation.client_operation_id) continue;
      if (item.operation_type !== "save_standalone_layout" || item.payload?.layout_key !== layoutKey) continue;
      if (item.status === "pending" || item.status === "failed") {
        item.entity_id = result.entity_id;
        item.base_version = result.version || item.base_version;
        await putRecord("operations", item);
      }
    }
    const form = document.querySelector("#layout-save-form[data-operation-type='save_standalone_layout']");
    if (form && form.dataset.layoutKey === layoutKey) {
      form.dataset.entityId = String(result.entity_id);
      form.dataset.version = String(result.version || form.dataset.version || 0);
      form.action = `/layouts/${result.entity_id}`;
      form.dataset.layoutKey = `standalone:${result.entity_id}`;
      const editor = document.querySelector("[data-layout-editor]");
      if (editor) editor.dataset.layoutId = String(result.entity_id);
    }
  }

  async function applyLayoutSyncResult(operation, result) {
    if (!["save_event_layout", "save_standalone_layout"].includes(operation.operation_type) || result.status !== "synced") return;
    const form = document.querySelector("#layout-save-form");
    if (!form || form.dataset.layoutKey !== operation.payload?.layout_key) return;
    if (result.version) form.dataset.version = String(result.version);
    form.dataset.offlineQueued = "";
    document.querySelectorAll(`button[form="${form.id}"]`).forEach((button) => {
      if (button.dataset.originalLabel) button.textContent = button.dataset.originalLabel;
    });
    await deleteLayoutDraft(form.dataset.layoutKey);
    if (operation.operation_type === "save_standalone_layout") {
      await applyStandaloneLayoutSyncResult(operation, result);
    }
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
      const checkIcon = window.emenysIcon ? window.emenysIcon("check") : "";
      actions.innerHTML = `<button type="button" class="button primary" data-quick-complete>${checkIcon} Concluído</button><button type="button" class="button danger" data-quick-missing>Falta quantidade</button><div class="mobile-missing-editor" hidden><label>Quanto falta?<input type="number" min="1" step="1" required></label><button type="button" class="button danger" data-quick-missing-save>Confirmar falta</button></div><small aria-live="polite"></small>`;
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

  async function syncOperations(force = false) {
    if (synchronizing) return;
    if (serviceReachable === false || (!navigator.onLine && serviceReachable !== true)) return;
    if (!force && !isSyncEnabled()) {
      await updateStatus();
      return;
    }
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
          if (result.status === "synced") await applyLayoutSyncResult(operation, result);
        }
      }
      await syncPhotos();
      await refreshBootstrap();
      renderConflicts();
      await updateStatus();
      const syncedLayouts = pending.filter((item) => ["save_event_layout", "save_standalone_layout"].includes(item.operation_type) && item.status === "synced");
      if (syncedLayouts.length && document.querySelector("[data-layout-editor]")) {
        showLayoutOfflineNotice("Layout sincronizado com o servidor.");
      }
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
          if (button.dataset.choice === "merge" && conflict.server_snapshot) {
            if (conflict.operation_type === "update_event_draft") {
              conflict.payload = { ...conflict.server_snapshot, ...conflict.payload };
            } else if (["save_event_layout", "save_standalone_layout"].includes(conflict.operation_type)) {
              conflict.base_version = conflict.server_version || conflict.base_version;
            }
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

  function eventField(event, ...keys) {
    for (const key of keys) {
      if (event?.[key] != null && event[key] !== "") return event[key];
    }
    return "";
  }

  const icon = (name) => (window.emenysIcon ? window.emenysIcon(name) : "");

  function fillDataIcons() {
    document.querySelectorAll("[data-icon]").forEach((node) => {
      node.innerHTML = icon(node.dataset.icon);
    });
  }

  function emptyState(title, copy) {
    return `<div class="empty-state panel">${icon("events")}<strong>${title}</strong><p>${copy}</p></div>`;
  }

  async function renderOfflineHome() {
    const root = document.querySelector("[data-offline-home]");
    if (!root) return;
    const saved = await getRecord("meta", "bootstrap").catch(() => null);
    if (!saved?.value) {
      root.innerHTML = emptyState("Nenhum evento foi salvo neste aparelho.", "Entre no modo online, abra as checklists e use “Salvar eventos neste aparelho”.");
      return;
    }
    const expiration = new Date(saved.value.offline_access_expires_at || 0);
    if (expiration < new Date()) {
      root.innerHTML = emptyState("O acesso offline expirou.", "Conecte-se novamente para validar sua sessão e baixar os eventos.");
      return;
    }
    root.replaceChildren();
    const events = saved.value.events || [];
    if (!events.length) {
      root.innerHTML = emptyState("Nenhum evento disponível offline.", "Cadastre um evento no modo online e salve-o neste aparelho.");
      return;
    }
    events.forEach((bundle) => {
      const event = bundle.event || {};
      const id = eventField(event, "id", "ID");
      const name = eventField(event, "name", "Name") || "Evento";
      const client = eventField(event, "client_name", "ClientName");
      const guests = eventField(event, "guest_count", "GuestCount");
      const venue = eventField(event, "venue", "Venue");
      const items = bundle.checklist?.items || bundle.checklist?.Items || [];
      const card = document.createElement("article");
      card.className = "panel offline-event-card";
      card.innerHTML = `<header><span class="offline-card-icon">${icon("events")}</span><div><h2></h2><p></p></div></header><p></p><div class="offline-event-actions"><a class="button primary" href="/events/${id}/operation">${icon("checklists")} Abrir checklist</a><a class="button secondary" href="/events/${id}/layout">${icon("layouts")} Organizar layout</a></div>`;
      card.querySelector("h2").textContent = name;
      card.querySelector("header p").textContent = [client, venue, guests ? `${guests} pessoas` : ""].filter(Boolean).join(" · ");
      card.querySelector("header + p").textContent = `${items.length} itens na checklist salva`;
      root.append(card);
    });
    const layouts = saved.value.standalone_layouts || saved.value.standaloneLayouts || [];
    if (layouts.length) {
      const block = document.createElement("section");
      block.className = "panel offline-layout-panel";
      block.innerHTML = `<div class="panel-heading"><div><span class="eyebrow">Organizador de layout</span><h2>Plantas salvas neste aparelho</h2></div></div><div data-layout-list></div>`;
      const list = block.querySelector("[data-layout-list]");
      layouts.forEach((layout) => {
        const id = eventField(layout, "id", "ID");
        const row = document.createElement("p");
        const link = document.createElement("a");
        link.className = "text-link";
        link.href = `/layouts/${id}`;
        link.innerHTML = `${eventField(layout, "name", "Name") || "Layout"} ${icon("arrow")}`;
        row.append(link);
        list.append(row);
      });
      root.append(block);
    }
  }

  function bindSyncPreference() {
    const toggle = document.querySelector("[data-sync-enabled]");
    if (toggle) {
      toggle.checked = isSyncEnabled();
      toggle.addEventListener("change", async () => {
        setSyncEnabled(toggle.checked);
        await updateStatus();
        if (toggle.checked && navigator.onLine) await syncOperations(true);
      });
    }
    document.querySelectorAll("[data-download-offline]").forEach((button) => {
      button.addEventListener("click", async () => {
        button.disabled = true;
        try {
          await refreshBootstrap();
          await renderOfflineHome();
          window.alert("Eventos e layouts foram salvos neste aparelho.");
        } catch (error) {
          window.alert(error.message || "Não foi possível salvar os dados offline.");
        } finally {
          button.disabled = false;
        }
      });
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

  window.BuffetFlowOffline = {
    saveLayoutDraft,
    loadLayoutDraft,
    deleteLayoutDraft,
    showLayoutOfflineNotice,
    syncOperations,
  };

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
    if (event.target.closest("[data-sync-now]")) syncOperations(true);
    if (event.target.closest("[data-close-conflicts]")) document.querySelector("[data-conflict-panel]").hidden = true;
    if (event.target.closest("[data-stay-offline]")) {
      reconnectDismissed = true;
      showReconnectBanner(false);
    }
    if (event.target.closest("[data-open-online]")) {
      event.preventDefault();
      openOnlineMode();
    }
  });
  window.addEventListener("online", () => watchServiceConnection());
  window.addEventListener("offline", () => applyServiceState(false));
  document.addEventListener("visibilitychange", () => {
    if (!document.hidden) watchServiceConnection();
  });
  navigator.serviceWorker?.addEventListener("message", (event) => {
    if (event.data?.type === "SYNC_REQUESTED") syncOperations();
  });
  document.addEventListener("DOMContentLoaded", async () => {
    if ("serviceWorker" in navigator) {
      registration = await navigator.serviceWorker.register("/sw.js").catch(() => null);
      registration?.waiting?.postMessage({ type: "SKIP_WAITING" });
      registration?.update().catch(() => null);
    }
    fillDataIcons();
    bindSyncPreference();
    await updateStatus();
    await renderConflicts();
    await renderOfflineHome();
    initializeOperationalQuickLoading();
    await watchServiceConnection();
    if (serviceReachable && isSyncEnabled()) await syncOperations(true);
    probeTimer = window.setInterval(watchServiceConnection, PROBE_MS);
  });
})();
