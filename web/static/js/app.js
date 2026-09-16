function systemLayer(className, content) {
  const layer = document.createElement("div");
  layer.className = `system-layer ${className || ""}`;
  layer.innerHTML = content;
  document.body.appendChild(layer);
  return layer;
}

function showSystemAlert(message, tone = "info") {
  let stack = document.querySelector(".system-toast-stack");
  if (!stack) {
    stack = document.createElement("div");
    stack.className = "system-toast-stack";
    stack.setAttribute("aria-live", "polite");
    document.body.appendChild(stack);
  }
  const toast = document.createElement("div");
  toast.className = `system-toast ${tone}`;
  toast.innerHTML = `<span></span><button type="button" aria-label="Fechar">×</button>`;
  toast.querySelector("span").textContent = message;
  const dismiss = () => toast.remove();
  toast.querySelector("button").addEventListener("click", dismiss);
  stack.appendChild(toast);
  window.setTimeout(dismiss, 5500);
}

function systemConfirm(message, options = {}) {
  return new Promise((resolve) => {
    const layer = systemLayer("system-confirm", `<section class="system-dialog" role="dialog" aria-modal="true" aria-labelledby="system-confirm-title"><h2 id="system-confirm-title">Confirmar ação</h2><p></p><div class="system-dialog-actions"><button class="button secondary small" type="button" data-system-cancel>Voltar</button><button class="button primary small" type="button" data-system-confirm>${options.confirmLabel || "Confirmar"}</button></div></section>`);
    layer.querySelector("p").textContent = message;
    const finish = (answer) => { layer.remove(); resolve(answer); };
    layer.querySelector("[data-system-cancel]").addEventListener("click", () => finish(false));
    layer.querySelector("[data-system-confirm]").addEventListener("click", () => finish(true));
    layer.addEventListener("click", (event) => { if (event.target === layer) finish(false); });
    layer.querySelector("[data-system-cancel]").focus();
  });
}

function showImagePreview(src, alt) {
  const layer = systemLayer("image-preview-layer", `<section class="system-dialog image-preview-dialog" role="dialog" aria-modal="true" aria-label="Pré-visualização da referência"><img><footer><span></span><button class="button secondary small" type="button">Voltar</button></footer></section>`);
  const image = layer.querySelector("img");
  image.src = src;
  image.alt = alt || "Referência";
  layer.querySelector("footer span").textContent = alt || "Referência";
  const close = () => layer.remove();
  layer.querySelector("button").addEventListener("click", close);
  layer.addEventListener("click", (event) => { if (event.target === layer) close(); });
  layer.querySelector("button").focus();
}

window.emenysAlert = showSystemAlert;
window.emenysConfirm = systemConfirm;

document.addEventListener("submit", (event) => {
  const form = event.target.closest("form");
  const message = event.submitter?.dataset.confirm || form?.dataset.confirm;
  if (form && message && form.dataset.systemConfirmed !== "true") {
    event.preventDefault();
    systemConfirm(message, { confirmLabel: event.submitter?.dataset.confirmLabel || "Confirmar" }).then((confirmed) => {
      if (!confirmed) return;
      form.dataset.systemConfirmed = "true";
      form.requestSubmit(event.submitter || undefined);
      delete form.dataset.systemConfirmed;
    });
  }

  const preservedForm = event.target.closest("form[data-preserve-scroll]");
  if (preservedForm && !event.defaultPrevented) {
    sessionStorage.setItem("buffetflow-preserved-scroll", JSON.stringify({ path: window.location.pathname, top: window.scrollY }));
    window.setTimeout(() => sessionStorage.removeItem("buffetflow-preserved-scroll"), 2000);
  }
}, true);

document.addEventListener("click",(event)=>{
  if(event.target.closest("[data-group-check], [data-group-check-all], [data-group-defer]"))event.stopPropagation();
  const removeChoice = event.target.closest("[data-remove-model-choice]");
  if (removeChoice) {
    const row = removeChoice.closest(".model-choice");
    if (!row) return;
    event.preventDefault();
    const checkbox = row.querySelector('input[name="model_item_ids"]');
    if (checkbox) {
      checkbox.checked = false;
      checkbox.disabled = true;
      row.hidden = true;
      return;
    }
    row.remove();
  }
});

function initializeChecklistSpreadsheetDownload(root = document) {
  root.querySelectorAll("[data-download-checklist-csv]").forEach((link) => {
    if (link.dataset.initialized === "true") return;
    link.dataset.initialized = "true";
    link.addEventListener("click", (event) => {
      event.preventDefault();
      const download = document.createElement("a");
      download.href = link.href;
      download.download = link.download;
      download.hidden = true;
      document.body.appendChild(download);
      download.click();
      download.remove();
    });
  });
}

function bindFilePreview(input) {
  if (!input || input.dataset.filePreviewInitialized === "true") return;
  input.dataset.filePreviewInitialized = "true";
  const container = input.closest(".decoration-reference-upload, .note-photo-upload, [data-photo-upload]") || input.parentElement;
  const status = container?.querySelector("[data-file-picker-status]");
  const previews = container?.querySelector("[data-file-previews]");
  input.addEventListener("change", () => {
      const files = Array.from(input.files || []);
      if (status) status.textContent = files.length ? `${files.length} foto${files.length === 1 ? "" : "s"} selecionada${files.length === 1 ? "" : "s"}` : "JPG, PNG ou WEBP · até 8 MB";
      if (!previews) return;
      previews.replaceChildren();
      previews.hidden = files.length === 0;
      files.slice(0, 4).forEach((file) => {
        const image = document.createElement("img");
        image.alt = `Prévia de ${file.name}`;
        image.src = URL.createObjectURL(file);
        image.addEventListener("load", () => URL.revokeObjectURL(image.src), { once: true });
        previews.appendChild(image);
      });
      if (files.length > 4) {
        const more = document.createElement("span");
        more.textContent = `+${files.length - 4} foto${files.length - 4 === 1 ? "" : "s"}`;
        previews.appendChild(more);
      }
  });
}

function initializePhotoInputs(root = document) {
  root.querySelectorAll("[data-photo-upload] input[type='file'], input[data-file-input]").forEach(bindFilePreview);
  root.querySelectorAll("[data-image-preview]").forEach((button) => {
    if (button.dataset.imagePreviewInitialized === "true") return;
    button.dataset.imagePreviewInitialized = "true";
    button.addEventListener("click", () => showImagePreview(button.dataset.previewSrc, button.dataset.previewAlt));
  });
}

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
  initializeSimpleChecklist(root);
}

function initializeSimpleChecklist(root = document) {
  root.querySelectorAll("[data-simple-checklist]").forEach((list) => {
    if (list.dataset.simpleInitialized === "true") return;
    list.dataset.simpleInitialized = "true";
    const cards = Array.from(list.querySelectorAll("[data-simple-item]"));
    const form = list.closest("form");
    const finalizeButton = form ? form.querySelector("[data-loading-finalize]") : null;
    const progress = list.querySelector("[data-loading-progress]");
    const group = list.closest("[data-checklist-group]");
    const checkAll = group?.querySelector("[data-group-check-all]");
    const deferButton = group?.querySelector("[data-group-defer]");
    const laterKey = () => `emenys-group-later:${list.dataset.eventId || "0"}:${list.dataset.stage || "separation"}:${group?.dataset.groupKey || ""}`;
    const pendingCards = () => cards.filter((card) => card.dataset.state === "pending" || card.dataset.state === "waiting");
    const refreshGroupActions = () => {
      if (checkAll) checkAll.disabled = pendingCards().length === 0;
    };

    const updateProgress = () => {
      const decided = cards.filter((card) => card.dataset.state === "done" || card.dataset.state === "missing" || card.dataset.state === "not-have").length;
      if (progress) progress.textContent = `${decided} de ${cards.length}`;
      if (finalizeButton && cards.length > 0) {
        finalizeButton.disabled = decided !== cards.length;
        finalizeButton.title = decided === cards.length ? "" : "Marque todos os itens antes de finalizar.";
      }
    };

    const cardLabel = (card, kind) => {
      if (kind === "missing") return card.dataset.missingLabel || "Sem estoque";
      if (kind === "waiting") return card.dataset.waitingLabel || "Aguardando";
      if (kind === "not-have") return card.dataset.notHaveLabel || "Não terá";
      return card.dataset.doneLabel || "Conferido";
    };

    const stateForAction = (kind) => {
      if (kind === "missing") return "missing";
      if (kind === "waiting") return "waiting";
      if (kind === "not-have") return "not-have";
      return "done";
    };

    const setCardState = (card, state, message) => {
      card.dataset.state = state;
      if (card.dataset.decision !== undefined) {
        card.dataset.decision = state === "done" ? "complete" : state === "missing" ? "missing" : "";
      }
      card.classList.toggle("is-done", state === "done");
      card.classList.toggle("is-missing", state === "missing");
      card.classList.toggle("is-waiting", state === "waiting");
      card.classList.toggle("is-not-have", state === "not-have");
      const status = card.querySelector("[data-simple-status]");
      if (status) {
        const labelKind = state === "done" ? "check" : state;
        status.textContent = message || (state === "pending" ? "" : cardLabel(card, labelKind));
      }
      const hiddenQuantity = form ? form.querySelector(`input[name="quantity_${card.dataset.itemId}"]`) : null;
      if (hiddenQuantity && state === "done") hiddenQuantity.value = card.dataset.required || hiddenQuantity.value;
      updateProgress();
      refreshGroupActions();
    };

    const postAction = async (card, kind) => {
      if (card.dataset.busy === "1") return;
      card.dataset.busy = "1";
      const status = card.querySelector("[data-simple-status]");
      if (status) status.textContent = navigator.onLine ? "Salvando…" : "Salvando no aparelho…";
      const required = card.dataset.required || "0";
      const eventID = Number(list.dataset.eventId || 0);
      const itemID = Number(card.dataset.itemId || 0);
      try {
        if (!navigator.onLine && typeof window.emenysQueueChecklistAction === "function") {
          await window.emenysQueueChecklistAction({
            mode: card.dataset.mode,
            kind,
            eventID,
            itemID,
            stage: card.dataset.stage,
            required,
            version: Number(card.dataset.version || 0),
          });
          setCardState(card, stateForAction(kind), `${cardLabel(card, kind)} — no aparelho`);
          return;
        }
        const submit = (versionValue) => {
          const body = new URLSearchParams();
          let url = kind === "missing" ? card.dataset.missingUrl : card.dataset.checkUrl;
          if (kind === "waiting" || kind === "not-have") {
            body.set("event_id", String(eventID));
            body.set("status", kind === "waiting" ? "pending" : "not_applicable");
            url = card.dataset.statusUrl;
          } else if (card.dataset.mode === "loading-decision") {
            body.set("decision", kind === "missing" ? "missing" : "complete");
            body.set("missing_quantity", kind === "missing" ? required : "0");
            url = card.dataset.saveUrl || url;
          } else if (kind === "missing") {
            body.set("missing_quantity", required);
            body.set("reason", "Não tem no estoque");
            body.set("resolution_type", "other");
          } else {
            body.set("stage", card.dataset.stage || "separation");
            body.set("quantity", required);
            body.set("version", versionValue);
          }
          return fetch(url, {
            method: "POST",
            body,
            credentials: "same-origin",
            headers: { Accept: "application/json", "X-BuffetFlow-Client": "pwa" },
          });
        };
        let response = await submit(card.dataset.version || "0");
        if (response.status === 409 && kind !== "missing" && kind !== "waiting" && kind !== "not-have") {
          response = await submit("0");
        }
        const result = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(result.error || "Não foi possível salvar.");
        if (result.version) card.dataset.version = String(result.version);
        setCardState(card, stateForAction(kind));
      } catch (error) {
        if (status) status.textContent = error.message || "Não foi possível salvar.";
      } finally {
        card.dataset.busy = "";
        resetSwipe(card);
      }
    };

    const resetSwipe = (card) => {
      const front = card.querySelector(".simple-check-front");
      if (front) {
        front.style.transition = "transform .18s ease";
        front.style.transform = "translateX(0)";
      }
      card.classList.remove("is-swiping-left", "is-swiping-right");
    };

    cards.forEach((card) => {
      const front = card.querySelector(".simple-check-front");
      const isActionControl = (target) => Boolean(target?.closest?.("button, [data-simple-check], [data-simple-missing]"));
      const checkButton = card.querySelector("[data-simple-check]");
      const missingButton = card.querySelector("[data-simple-missing]");
      const waitingButton = card.querySelector("[data-simple-waiting]");
      const notHaveButton = card.querySelector("[data-simple-not-have]");
      const bindAction = (button, kind) => {
        if (!button) return;
        button.addEventListener("pointerdown", (event) => event.stopPropagation());
        button.addEventListener("click", (event) => {
          event.preventDefault();
          event.stopPropagation();
          postAction(card, kind);
        });
      };
      bindAction(checkButton, "check");
      bindAction(missingButton, "missing");
      bindAction(waitingButton, "waiting");
      bindAction(notHaveButton, "not-have");
      if (!front) return;

      let pointer = null;
      const startSwipe = (event) => {
        if (event.pointerType === "mouse" && event.button !== 0) return;
        if (isActionControl(event.target)) return;
        pointer = {
          id: event.pointerId,
          x: event.clientX,
          y: event.clientY,
          locked: false,
        };
        front.style.transition = "none";
        front.setPointerCapture?.(event.pointerId);
      };
      const moveSwipe = (event) => {
        if (!pointer || event.pointerId !== pointer.id) return;
        const dx = event.clientX - pointer.x;
        const dy = event.clientY - pointer.y;
        if (!pointer.locked) {
          if (Math.abs(dx) < 8 && Math.abs(dy) < 8) return;
          if (Math.abs(dy) > Math.abs(dx)) {
            pointer = null;
            return;
          }
          pointer.locked = true;
          card.dataset.didSwipe = "1";
        }
        event.preventDefault();
        const x = Math.max(-140, Math.min(140, dx));
        front.style.transform = `translateX(${x}px)`;
        card.classList.toggle("is-swiping-right", x > 24);
        card.classList.toggle("is-swiping-left", x < -24);
      };
      const endSwipe = (event) => {
        if (!pointer || event.pointerId !== pointer.id) return;
        const dx = event.clientX - pointer.x;
        const swiped = pointer.locked;
        pointer = null;
        if (dx >= 72) {
          postAction(card, "check");
          return;
        }
        if (dx <= -72) {
          postAction(card, card.dataset.stage === "separation" ? "not-have" : "missing");
          return;
        }
        resetSwipe(card);
        if (!swiped) card.dataset.didSwipe = "";
      };

      front.addEventListener("pointerdown", startSwipe);
      front.addEventListener("pointermove", moveSwipe);
      front.addEventListener("pointerup", endSwipe);
      front.addEventListener("pointercancel", endSwipe);
      front.addEventListener("click", (event) => {
        if (isActionControl(event.target)) return;
        if (card.dataset.didSwipe !== "1") return;
        event.preventDefault();
        event.stopPropagation();
        card.dataset.didSwipe = "";
      }, true);
    });
    if (group?.tagName === "DETAILS") {
      try {
        if (sessionStorage.getItem(laterKey()) === "1") group.open = false;
      } catch (_error) {
        // Ignore storage access errors in private browsing.
      }
    }
    if (checkAll) {
      checkAll.addEventListener("click", async (event) => {
        event.preventDefault();
        event.stopPropagation();
        const pending = pendingCards();
        if (pending.length === 0) return;
        const groupName = group?.querySelector(".checklist-group-heading h2")?.textContent.trim() || "esta classe";
        const actionLabel = /separar/i.test(checkAll.textContent) ? "Separar" : "Conferir";
        const itemLabel = pending.length === 1 ? "item" : "itens";
        if (!await systemConfirm(`${actionLabel} ${pending.length} ${itemLabel} de ${groupName}?`, { confirmLabel: actionLabel })) return;
        checkAll.disabled = true;
        try {
          sessionStorage.removeItem(laterKey());
        } catch (_error) {
          // Ignore storage access errors in private browsing.
        }
        if (group?.tagName === "DETAILS") group.open = true;
        for (const card of pending) {
          await postAction(card, "check");
        }
        refreshGroupActions();
      });
    }
    if (deferButton) {
      deferButton.addEventListener("click", (event) => {
        event.preventDefault();
        event.stopPropagation();
        if (group?.tagName === "DETAILS") group.open = false;
        try {
          sessionStorage.setItem(laterKey(), "1");
        } catch (_error) {
          // Ignore storage access errors in private browsing.
        }
      });
    }
    updateProgress();
    refreshGroupActions();
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
    const hasSavedData = () => section.querySelector('input[type="checkbox"]:checked');
    toggle.addEventListener("change", async () => {
      if (!toggle.checked && hasSavedData() && !await systemConfirm("Desativar a decoração? As peças escolhidas serão preservadas, mas não entrarão na checklist.")) toggle.checked = true;
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

function initializeCustomMenuItems(root = document) {
  root.querySelectorAll("[data-custom-menu-items]").forEach((editor) => {
    if (editor.dataset.customMenuItemsInitialized === "true") return;
    editor.dataset.customMenuItemsInitialized = "true";
    const template = editor.querySelector("[data-custom-menu-item-template]");
    const addButton = editor.querySelector("[data-add-custom-menu-item]");
    if (!template || !addButton) return;

    addButton.addEventListener("click", () => {
      const row = template.content.firstElementChild?.cloneNode(true);
      if (!row) return;
      addButton.before(row);
      const input = row.querySelector(".model-item-name-input");
      input?.focus();
      row.dataset.fixedChoiceBound = "true";
      row.addEventListener("click", (event) => {
        if (event.target.closest("button, input, a")) return;
        input?.focus();
      });
    });
  });

  root.querySelectorAll(".model-choice.fixed").forEach((row) => {
    if (row.dataset.fixedChoiceBound === "true") return;
    row.dataset.fixedChoiceBound = "true";
    row.addEventListener("click", (event) => {
      if (event.target.closest("button, input, a")) return;
      row.querySelector(".model-item-name-input")?.focus();
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

function initializeEventVenueName(root = document) {
  root.querySelectorAll('form.form-layout input[name="venue"]').forEach((venue) => {
    if (venue.dataset.venueNameBound === "1") return;
    venue.dataset.venueNameBound = "1";
    const form = venue.closest("form");
    const name = form?.querySelector('input[name="name"]');
    const sync = () => {
      if (name) name.value = venue.value.trim();
    };
    venue.addEventListener("input", sync);
    venue.addEventListener("change", sync);
    sync();
  });
}

function initializeCalendarDatePicker(root = document) {
  const selectionPanel = root.querySelector("[data-calendar-selection-panel]");
  const templateFor = (date) => Array.from(root.querySelectorAll("[data-calendar-selection-template]")).find((template) => template.dataset.calendarSelectionTemplate === date);
  const selectDate = (day, updateHistory = true) => {
    if (!selectionPanel) return;
    const template = templateFor(day.dataset.calendarDate);
    if (!template) return;

    selectionPanel.replaceChildren(template.content.cloneNode(true));
    root.querySelectorAll("[data-calendar-date]").forEach((candidate) => {
      const selected = candidate === day;
      candidate.classList.toggle("is-selected", selected);
      candidate.setAttribute("aria-pressed", String(selected));
    });

    if (updateHistory) {
      const url = new URL(window.location.href);
      url.searchParams.set("month", day.dataset.calendarMonth);
      url.searchParams.set("date", day.dataset.calendarDate);
      window.history.replaceState({}, "", url);
    }
  };

  root.querySelectorAll("[data-calendar-date]").forEach((day) => {
    if (day.dataset.calendarBound === "true") return;
    day.dataset.calendarBound = "true";
    day.addEventListener("click", (event) => {
      event.preventDefault();
      selectDate(day);
    });
    day.addEventListener("dblclick", async (event) => {
      event.preventDefault();
      const target = day.dataset.createUrl;
      if (!target) return;
      const date = day.dataset.calendarDate.split("-").reverse().join("/");
      if (await systemConfirm(`Deseja criar um evento para ${date}?`, { confirmLabel: "Criar evento" })) window.location.assign(target);
    });

    if (day.classList.contains("is-selected")) selectDate(day, false);
  });
}

function watchMenuModelPreview(root = document) {
  const preview = root.querySelector?.("#menu-model-preview") || document.getElementById("menu-model-preview");
  if (!preview || preview.dataset.customMenuObserver === "true") return;
  preview.dataset.customMenuObserver = "true";
  new MutationObserver(() => initializeCustomMenuItems(document)).observe(preview, { childList: true, subtree: true });
}

document.addEventListener("DOMContentLoaded", () => {
	restorePreservedScroll();
  updatePrimaryNavigation();
  initializeMenuTemplateSelectors();
  initializeMenuModelFallback();
  initializeChecklistSpreadsheetDownload();
  initializePDFSharing();
  initializeMobileLoading();
	initializeMenuCategoryRules();
	initializeEventDecorationToggle();
	initializeEventCakeToggle();
	initializeRentedDecorations();
	initializeChecklistObservations();
	initializeCustomMenuItems();
	watchMenuModelPreview();
	initializeInventoryInternalCode();
	initializeEventVenueName();
	initializeCalendarDatePicker();
	initializePhotoInputs();
});

document.addEventListener("htmx:beforeRequest", (event) => {
  const link = event.detail.elt?.closest?.(".nav-list a, .mobile-nav a:not(.mobile-create)");
  if (link) updatePrimaryNavigation(new URL(link.href, window.location.origin).pathname);
});

document.addEventListener("htmx:pushedIntoHistory", () => {
  updatePrimaryNavigation();
});

function refreshAfterSwap(root = document) {
  updatePrimaryNavigation();
  initializeMenuTemplateSelectors(root);
  initializeMenuModelFallback(root);
  initializeChecklistSpreadsheetDownload(root);
  initializePDFSharing(root);
  initializeMobileLoading(root);
  initializeMenuCategoryRules(root);
  initializeEventDecorationToggle(root);
  initializeEventCakeToggle(document);
  initializeRentedDecorations(root);
  initializeChecklistObservations(root);
  initializeCustomMenuItems(document);
  watchMenuModelPreview(document);
  initializeInventoryInternalCode(root);
  initializeEventVenueName(root);
  initializeCalendarDatePicker(root);
  initializePhotoInputs(root);
}

document.addEventListener("htmx:afterSwap", (event) => {
  refreshAfterSwap(event.detail?.target || event.target || document);
}, true);
document.addEventListener("htmx:afterSettle", () => {
  initializeCustomMenuItems(document);
}, true);
