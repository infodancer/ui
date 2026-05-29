/*
 * mdedit.js — the loader and adapter seam.
 *
 * The seam: every editor we might use (EasyMDE today; CodeMirror 6 or
 * Toast UI later) is wrapped by an *adapter* that enhances a plain
 * <textarea> and keeps the textarea's value authoritative. The textarea is
 * what htmx/forms serialize, so the rest of the stack neither knows nor
 * cares which editor is mounted. Swapping editors is choosing a different
 * adapter — no server, template, or htmx change.
 *
 * Adapter contract — register with mdedit.register(name, mount):
 *
 *   mount(textarea, opts) -> controller
 *
 *   opts = { livePreview: bool, toolbar: "minimal"|"standard"|"full" }
 *   controller = {
 *     getValue(): string        // current editor content
 *     sync(): void              // flush editor content into textarea.value
 *     destroy(): void           // tear down, restore the bare textarea
 *     onChange(cb): void        // optional; cb() on each edit (for live preview)
 *   }
 *
 * Responsibilities the loader owns (so adapters stay thin):
 *   - enhancing [data-mdedit] textareas on load and after htmx swaps
 *   - flushing every editor's value into its textarea before any htmx
 *     request, so Save/Preview serialize current content
 *   - tearing down editors when htmx removes their element
 *   - wiring debounced server-side live preview when data-mdedit-live is set
 */
(function () {
  "use strict";

  var registry = Object.create(null);
  var DEFAULT_ADAPTER = "easymde";

  // Reuse any window.mdedit an adapter created before this file ran, so load
  // order between the loader and its adapters does not matter.
  var mdedit = window.mdedit || {};
  mdedit.register = function (name, mount) {
    registry[name] = mount;
  };
  window.mdedit = mdedit;

  // Drain registrations queued by adapters that loaded first.
  if (mdedit._pending) {
    for (var p = 0; p < mdedit._pending.length; p++) {
      registry[mdedit._pending[p][0]] = mdedit._pending[p][1];
    }
    mdedit._pending = null;
  }

  function enhance(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var nodes = scope.querySelectorAll("textarea[data-mdedit]");
    for (var i = 0; i < nodes.length; i++) {
      mountOne(nodes[i]);
    }
    // querySelectorAll on an element excludes the element itself.
    if (root && root.matches && root.matches("textarea[data-mdedit]")) {
      mountOne(root);
    }
  }

  function mountOne(ta) {
    if (ta._mdeditController) return; // already enhanced
    var name = ta.getAttribute("data-mdedit-adapter") || DEFAULT_ADAPTER;
    var mount = registry[name];
    if (!mount) {
      // No adapter (script failed to load, unknown name): leave the plain
      // textarea. It still works — degraded, not broken.
      console.warn("mdedit: no adapter registered for", name, "— using plain textarea");
      return;
    }
    var live = ta.getAttribute("data-mdedit-live") === "1";
    var toolbar = ta.getAttribute("data-mdedit-toolbar") || "full";
    var controller = mount(ta, { livePreview: live, toolbar: toolbar });
    ta._mdeditController = controller;

    if (live && controller.onChange) {
      var previewURL = ta.getAttribute("data-mdedit-preview-url");
      var target = ta.getAttribute("data-mdedit-preview-target");
      if (previewURL && target && window.htmx) {
        var debounced = debounce(function () {
          controller.sync();
          window.htmx.ajax("POST", previewURL, {
            target: target,
            swap: "innerHTML",
            values: { markdown: controller.getValue() },
          });
        }, 500);
        controller.onChange(debounced);
      }
    }
  }

  function teardown(el) {
    var nodes =
      el.querySelectorAll ? el.querySelectorAll("textarea[data-mdedit]") : [];
    for (var i = 0; i < nodes.length; i++) {
      if (nodes[i]._mdeditController) {
        nodes[i]._mdeditController.destroy();
        nodes[i]._mdeditController = null;
      }
    }
    if (el._mdeditController) {
      el._mdeditController.destroy();
      el._mdeditController = null;
    }
  }

  function syncAll() {
    var nodes = document.querySelectorAll("textarea[data-mdedit]");
    for (var i = 0; i < nodes.length; i++) {
      if (nodes[i]._mdeditController) nodes[i]._mdeditController.sync();
    }
  }

  function debounce(fn, ms) {
    var t;
    return function () {
      clearTimeout(t);
      t = setTimeout(fn, ms);
    };
  }

  // Initial page load (no htmx) and progressive enhancement.
  document.addEventListener("DOMContentLoaded", function () {
    enhance(document);
  });

  // htmx lifecycle: enhance swapped-in content, flush before requests so
  // serialized values are current, and tear down before content is removed.
  document.body.addEventListener("htmx:load", function (e) {
    enhance(e.target);
  });
  document.body.addEventListener("htmx:configRequest", function () {
    syncAll();
  });
  document.body.addEventListener("htmx:beforeCleanupElement", function (e) {
    teardown(e.target);
  });
})();
