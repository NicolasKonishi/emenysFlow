document.addEventListener("submit", (event) => {
  const form = event.target.closest("form[data-confirm]");
  if (form && !window.confirm(form.dataset.confirm)) event.preventDefault();

  const preservedForm = event.target.closest("form[data-preserve-scroll]");
  if (preservedForm && !event.defaultPrevented) {
    sessionStorage.setItem("buffetflow-preserved-scroll", JSON.stringify({ path: window.location.pathname, top: window.scrollY }));
    window.setTimeout(() => sessionStorage.removeItem("buffetflow-preserved-scroll"), 2000);
  }
});

document.addEventListener("click",(event)=>{
  if(event.target.closest("[data-print]"))window.print();
  if(event.target.closest("[data-group-check]"))event.stopPropagation();
});

function restorePreservedScroll() {
  const raw = sessionStorage.getItem("buffetflow-preserved-scroll");
  if (!raw) return;
  sessionStorage.removeItem("buffetflow-preserved-scroll");
  try {
    const saved = JSON.parse(raw);
    if (saved.path === window.location.pathname && Number.isFinite(saved.top)) {
      window.requestAnimationFrame(() => window.scrollTo({ top: saved.top, behavior: "instant" }));
    }
  } catch (_error) {
    // Ignore an invalid value left by an older browser session.
  }
}

function navKeyForPath(pathname) {
  if (pathname === "/") return "dashboard";
  const segments = pathname.split("/").filter(Boolean);
  return segments[0] || "dashboard";
}

function updatePrimaryNavigation(pathname = window.location.pathname) {
  const current = navKeyForPath(pathname);
  document.querySelectorAll(".nav-list a, .mobile-nav a:not(.mobile-create)").forEach((link) => {
    const target = navKeyForPath(new URL(link.href, window.location.origin).pathname);
    if (target === current) {
      link.setAttribute("aria-current", "page");
    } else {
      link.removeAttribute("aria-current");
    }
  });
}

function initializeMenuTemplateSelectors(root = document) {
  root.querySelectorAll("[data-menu-template-select]").forEach((templateSelect) => {
    if (templateSelect.dataset.initialized === "true") return;
    templateSelect.dataset.initialized = "true";
    const form = templateSelect.closest("form");
    if (!form) return;
    const menuSelects = form.querySelectorAll('select[name="menu_item_ids"]');
    const customized = form.querySelector("#menu-customized");
    const status = form.querySelector("#template-applied-status");

    const updateAvailableItems = (templateID, preserveSelected) => {
      const copiedSourceIDs = new Set();
      menuSelects.forEach((menuSelect) => Array.from(menuSelect.options).forEach((itemOption) => {
        if (templateID && itemOption.dataset.templateOwner === templateID && itemOption.dataset.sourceItem) {
          copiedSourceIDs.add(itemOption.dataset.sourceItem);
        }
      }));
      menuSelects.forEach((menuSelect) => {
        Array.from(menuSelect.options).forEach((itemOption) => {
          const owner = itemOption.dataset.templateOwner || "";
          const duplicatedGlobal = !owner && copiedSourceIDs.has(itemOption.value);
          const visible = (!owner && !duplicatedGlobal) || owner === templateID || (preserveSelected && itemOption.selected);
          itemOption.hidden = !visible;
          itemOption.disabled = !visible;
          if (!visible && !preserveSelected) itemOption.selected = false;
        });
      });
    };

    updateAvailableItems(templateSelect.value, true);

    menuSelects.forEach((menuSelect) => menuSelect.addEventListener("change", () => {
      if (customized) customized.value = "1";
      if (status) status.textContent = "Cardápio personalizado para este evento.";
    }));

    templateSelect.addEventListener("change", () => {
      const option = templateSelect.selectedOptions[0];
      if (!option || !option.value) {
        updateAvailableItems("", true);
        if (customized) customized.value = "1";
        if (status) status.textContent = "Sem modelo vinculado. A seleção atual foi mantida.";
        return;
      }
      const selectedIDs = new Set((option.dataset.menuItems || "").split(",").filter(Boolean));
      updateAvailableItems(option.value, false);
      menuSelects.forEach((menuSelect) => {
        Array.from(menuSelect.options).forEach((itemOption) => {
          itemOption.selected = selectedIDs.has(itemOption.value);
        });
      });
      const decoration = form.querySelector("#event-has-decoration");
      const welcome = form.querySelector("#event-has-welcome-drinks");
      const coffee = form.querySelector("#event-has-coffee-table");
      if (decoration) decoration.checked = option.dataset.decoration === "true";
      if (welcome) welcome.checked = option.dataset.welcome === "true";
      if (coffee) coffee.checked = option.dataset.coffee === "true";
      if (customized) customized.value = "0";
      if (status) status.textContent = `Cardápio “${option.textContent.split(" — ")[0]}” aplicado. Você pode alterar os itens abaixo.`;
    });
  });
}

function initializeMenuModelFallback(root = document) {
  if (window.htmx) return;
  root.querySelectorAll('select[name="menu_model_id"][hx-get]').forEach((select) => {
    if (select.dataset.fallbackInitialized === "true") return;
    select.dataset.fallbackInitialized = "true";
    select.addEventListener("change", async () => {
      const target = document.querySelector(select.getAttribute("hx-target"));
      if (!target) return;
      select.disabled = true;
      try {
        const url = new URL(select.getAttribute("hx-get"), window.location.origin);
        url.searchParams.set(select.name, select.value);
        const response = await fetch(url, { headers: { "HX-Request": "true" }, credentials: "same-origin" });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        target.innerHTML = await response.text();
        initializeEventCakeToggle(document);
      } catch (_error) {
        target.innerHTML = '<div class="alert danger">Não foi possível carregar este modelo. Tente novamente.</div>';
      } finally {
        select.disabled = false;
      }
    });
  });
}

const pdfSharePayloads = new WeakMap();

function downloadPreparedPDF(payload) {
  const objectURL = URL.createObjectURL(payload.blob);
  const link = document.createElement("a");
  link.href = objectURL;
  link.download = payload.name;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(objectURL), 1000);
}

function openWhatsAppPDFDownload(payload, status) {
  downloadPreparedPDF(payload);
  const message = `O PDF “${payload.title}” foi baixado. Anexe o arquivo ${payload.name} nesta conversa.`;
  const whatsapp = window.open(`https://wa.me/?text=${encodeURIComponent(message)}`, "_blank", "noopener,noreferrer");
  status.classList.remove("error");
  status.classList.add("success");
  status.textContent = whatsapp
    ? "PDF baixado e WhatsApp aberto. Agora é só anexar o arquivo na conversa."
    : "PDF baixado. Abra o WhatsApp e anexe o arquivo na conversa.";
}

function initializePDFSharing(root = document) {
  root.querySelectorAll("[data-share-pdf]").forEach((button) => {
    if (button.dataset.initialized === "true") return;
    button.dataset.initialized = "true";
    const status = root.querySelector("[data-pdf-share-status]") || document.querySelector("[data-pdf-share-status]");
    const originalLabel = button.textContent;
    button.disabled = true;
    button.textContent = "Preparando PDF…";

    fetch(button.dataset.pdfUrl, { credentials: "same-origin" })
      .then((response) => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.blob();
      })
      .then((blob) => {
        const payload = { blob, name: button.dataset.pdfName, title: button.dataset.shareTitle, file: null };
        if (typeof File !== "undefined") payload.file = new File([blob], payload.name, { type: "application/pdf" });
        pdfSharePayloads.set(button, payload);
        button.disabled = false;
        button.textContent = originalLabel;
      })
      .catch(() => {
        button.textContent = "Compartilhamento indisponível";
        if (status) {
          status.classList.add("error");
          status.textContent = "Não foi possível preparar o compartilhamento. O download continua disponível.";
        }
      });

    button.addEventListener("click", async () => {
      const payload = pdfSharePayloads.get(button);
      if (!payload || !status) return;
      const canShareFile = payload.file && navigator.share && navigator.canShare && navigator.canShare({ files: [payload.file] });
      if (!canShareFile) {
        openWhatsAppPDFDownload(payload, status);
        return;
      }
      try {
        status.classList.remove("error", "success");
        status.textContent = "Escolha o WhatsApp no menu de compartilhamento do aparelho.";
        await navigator.share({ files: [payload.file], title: payload.title, text: "Checklist do evento em PDF" });
        status.classList.add("success");
        status.textContent = "PDF compartilhado.";
      } catch (error) {
        if (error && error.name === "AbortError") {
          status.textContent = "Compartilhamento cancelado. O PDF continua aberto abaixo.";
          return;
        }
        openWhatsAppPDFDownload(payload, status);
      }
    });
  });
}

function initializeMobileLoading(root = document) {
  if (!window.matchMedia("(max-width: 820px)").matches) return;
  root.querySelectorAll("[data-mobile-loading-list]").forEach((list) => {
    if (list.dataset.initialized === "true") return;
    list.dataset.initialized = "true";
    const cards = Array.from(list.querySelectorAll("[data-loading-item]"));
    const form = list.closest("form");
    const finalizeButton = form ? form.querySelector("[data-loading-finalize]") : null;
    const progress = list.querySelector("[data-loading-progress]");
    const formatter = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 2 });

    const updateProgress = () => {
      const decided = cards.filter((card) => card.dataset.decision === "complete" || card.dataset.decision === "missing").length;
      if (progress) progress.textContent = `${decided} de ${cards.length}`;
      if (finalizeButton && cards.length > 0) {
        finalizeButton.disabled = decided !== cards.length;
        finalizeButton.title = decided === cards.length ? "" : "Marque todos os itens antes de finalizar.";
      }
    };

    const saveDecision = async (card, decision, missingQuantity) => {
      const errorMessage = card.querySelector("[data-loading-error]");
      const controls = card.querySelectorAll("button, input");
      controls.forEach((control) => { control.disabled = true; });
      if (errorMessage) errorMessage.textContent = "Salvando…";
      try {
        const body = new URLSearchParams({ decision, missing_quantity: String(missingQuantity || 0) });
        const response = await fetch(card.dataset.saveUrl, { method: "POST", body, credentials: "same-origin" });
        const result = await response.json();
        if (!response.ok) throw new Error(result.error || "Não foi possível salvar.");

        card.dataset.decision = result.decision;
        card.classList.toggle("is-complete", result.decision === "complete");
        card.classList.toggle("has-missing", result.decision === "missing");
        const state = card.querySelector("[data-loading-state]");
        if (state) state.textContent = result.decision === "complete" ? "✓ Concluído" : `Falta ${formatter.format(result.missing_quantity)}`;
        const editor = card.querySelector("[data-loading-missing-editor]");
        if (editor) editor.hidden = true;
        const hiddenQuantity = form ? form.querySelector(`input[name="quantity_${card.dataset.itemId}"]`) : null;
        if (hiddenQuantity) hiddenQuantity.value = result.loaded_quantity;
        if (errorMessage) errorMessage.textContent = "Salvo";
        updateProgress();
      } catch (error) {
        if (errorMessage) errorMessage.textContent = error.message || "Não foi possível salvar. Tente novamente.";
      } finally {
        controls.forEach((control) => { control.disabled = false; });
      }
    };

    cards.forEach((card) => {
      const completeButton = card.querySelector("[data-loading-complete]");
      const missingToggle = card.querySelector("[data-loading-missing-toggle]");
      const missingEditor = card.querySelector("[data-loading-missing-editor]");
      const missingInput = card.querySelector("[data-loading-missing-quantity]");
      const missingConfirm = card.querySelector("[data-loading-missing-confirm]");
      if (completeButton) completeButton.addEventListener("click", () => saveDecision(card, "complete", 0));
      if (missingToggle && missingEditor) missingToggle.addEventListener("click", () => {
        missingEditor.hidden = !missingEditor.hidden;
        if (!missingEditor.hidden && missingInput) missingInput.focus();
      });
      if (missingConfirm && missingInput) missingConfirm.addEventListener("click", () => {
        const missing = Number.parseFloat(missingInput.value);
        const required = Number.parseFloat(card.dataset.required);
        const errorMessage = card.querySelector("[data-loading-error]");
        if (!Number.isFinite(missing) || missing <= 0 || missing > required) {
          if (errorMessage) errorMessage.textContent = `Informe uma falta entre 0 e ${formatter.format(required)}.`;
          return;
        }
        saveDecision(card, "missing", missing);
      });
    });
    updateProgress();
  });
}

function initializeMenuCategoryRules(root = document) {
  root.querySelectorAll("[data-menu-item-form]").forEach((form) => {
    if (form.dataset.categoryRulesInitialized === "true") return;
    form.dataset.categoryRulesInitialized = "true";
    const category = form.querySelector("[data-menu-category]");
    const settings = form.querySelector("[data-container-settings]");
    if (!category || !settings) return;
    const refresh = () => {
      const slug = category.selectedOptions[0]?.dataset.categorySlug || "";
      const automaticPan = slug === "main_courses" || slug === "sides";
      settings.querySelectorAll('select[name="container_type_id"], input[name="container_capacity"]').forEach((control) => {
        control.disabled = automaticPan;
        control.closest("label").hidden = automaticPan;
      });
      settings.classList.toggle("automatic-pan-category", automaticPan);
    };
    category.addEventListener("change", refresh);
    refresh();
  });
}

function initializeEventDecorationToggle(root = document) {
  root.querySelectorAll("#event-has-decoration").forEach((toggle) => {
    if (toggle.dataset.decorationInitialized === "true") return;
    toggle.dataset.decorationInitialized = "true";
    const form = toggle.closest("form");
    const section = form?.querySelector("[data-event-decoration-section]");
    if (!section) return;
    const hasSavedData = section.querySelector('input[name="decoration_theme"]')?.value
      || section.querySelector('textarea[name="decoration_description"]')?.value
      || section.querySelector('input[name="decoration_ids"]:checked');
    toggle.addEventListener("change", () => {
      if (!toggle.checked && hasSavedData && !window.confirm("Desativar a decoração? Os dados preenchidos serão preservados, mas itens e reservas deixarão de ser gerados.")) {
        toggle.checked = true;
      }
      section.hidden = !toggle.checked;
    });
  });
}

function refreshEventCakeOption(toggle) {
  const form = toggle.closest("form");
  if (!form) return;
  const flavorField = form.querySelector("[data-cake-flavor-field]");
  if (flavorField) {
    flavorField.hidden = !toggle.checked;
    flavorField.querySelectorAll("input").forEach((input) => { input.disabled = !toggle.checked; });
  }
  form.querySelectorAll("[data-cake-model-section]").forEach((section) => {
    section.hidden = !toggle.checked;
    section.querySelectorAll("input,select,textarea").forEach((control) => { control.disabled = !toggle.checked; });
    if (toggle.checked) section.querySelectorAll('input[name="model_item_ids"][data-model-included]').forEach((input) => { input.checked = true; });
  });
}

function initializeEventCakeToggle(root = document) {
  root.querySelectorAll("#event-has-cake").forEach((toggle) => {
    if (toggle.dataset.cakeInitialized !== "true") {
      toggle.dataset.cakeInitialized = "true";
      toggle.addEventListener("change", () => refreshEventCakeOption(toggle));
    }
    refreshEventCakeOption(toggle);
  });
}

function initializeRentedDecorations(root = document) {
  root.querySelectorAll("[data-rented-decoration-editor]").forEach((editor) => {
    if (editor.dataset.rentedDecorationInitialized === "true") return;
    editor.dataset.rentedDecorationInitialized = "true";
    const list = editor.querySelector("[data-rented-decoration-list]");
    const template = editor.querySelector("[data-rented-decoration-template]");
    const addButton = editor.querySelector("[data-add-rented-decoration]");
    if (!list || !template || !addButton) return;

    addButton.addEventListener("click", () => {
      const row = template.content.firstElementChild?.cloneNode(true);
      if (!row) return;
      list.appendChild(row);
      row.querySelector('input[name="rented_decoration_name"]')?.focus();
    });

    editor.addEventListener("click", (event) => {
      const removeButton = event.target.closest("[data-remove-rented-decoration]");
      if (!removeButton) return;
      const row = removeButton.closest(".rented-decoration-row");
      if (!row) return;
      const rows = list.querySelectorAll(".rented-decoration-row");
      if (rows.length > 1) {
        row.remove();
        return;
      }
      row.querySelectorAll("input").forEach((input) => { input.value = ""; });
    });
  });
}

function initializeChecklistObservations(root = document) {
  root.querySelectorAll("[data-checklist-observations-editor]").forEach((editor) => {
    if (editor.dataset.checklistObservationsInitialized === "true") return;
    editor.dataset.checklistObservationsInitialized = "true";
    const list = editor.querySelector("[data-checklist-observations-list]");
    const template = editor.querySelector("[data-checklist-observation-template]");
    const addButton = editor.querySelector("[data-add-checklist-observation]");
    if (!list || !template || !addButton) return;

    addButton.addEventListener("click", () => {
      const row = template.content.firstElementChild?.cloneNode(true);
      if (!row) return;
      list.appendChild(row);
      row.querySelector('input[name="checklist_observations"]')?.focus();
    });

    editor.addEventListener("click", (event) => {
      const removeButton = event.target.closest("[data-remove-checklist-observation]");
      if (!removeButton) return;
      removeButton.closest(".checklist-observation-row")?.remove();
    });
  });
}

function normalizeInventoryCodePart(value) {
  return value
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function initializeInventoryInternalCode(root = document) {
  root.querySelectorAll('[data-inventory-item-form][data-inventory-code-mode="create"]').forEach((form) => {
    if (form.dataset.inventoryCodeInitialized === "true") return;
    form.dataset.inventoryCodeInitialized = "true";
    const name = form.querySelector("[data-inventory-item-name]");
    const category = form.querySelector("[data-inventory-category]");
    const internalCode = form.querySelector("[data-inventory-internal-code]");
    if (!name || !category || !internalCode) return;

    const refresh = () => {
      const prefix = normalizeInventoryCodePart(category.selectedOptions[0]?.dataset.codePrefix || "").toUpperCase();
      const itemName = normalizeInventoryCodePart(name.value);
      internalCode.value = [prefix, itemName].filter(Boolean).join("-");
    };

    name.addEventListener("input", refresh);
    category.addEventListener("change", refresh);
    refresh();
  });
}

document.addEventListener("DOMContentLoaded", () => {
	restorePreservedScroll();
  updatePrimaryNavigation();
  initializeMenuTemplateSelectors();
  initializeMenuModelFallback();
  initializePDFSharing();
  initializeMobileLoading();
	initializeMenuCategoryRules();
	initializeEventDecorationToggle();
	initializeEventCakeToggle();
	initializeRentedDecorations();
	initializeChecklistObservations();
	initializeInventoryInternalCode();
});

document.addEventListener("htmx:beforeRequest", (event) => {
  const link = event.detail.elt?.closest?.(".nav-list a, .mobile-nav a:not(.mobile-create)");
  if (link) updatePrimaryNavigation(new URL(link.href, window.location.origin).pathname);
});

document.addEventListener("htmx:pushedIntoHistory", () => {
  updatePrimaryNavigation();
});

document.addEventListener("htmx:afterSwap", (event) => {
  updatePrimaryNavigation();
  initializeMenuTemplateSelectors(event.target);
  initializeMenuModelFallback(event.target);
  initializePDFSharing(event.target);
  initializeMobileLoading(event.target);
	initializeMenuCategoryRules(event.target);
	initializeEventDecorationToggle(event.target);
	initializeEventCakeToggle(document);
	initializeRentedDecorations(event.target);
	initializeChecklistObservations(event.target);
	initializeInventoryInternalCode(event.target);
});
