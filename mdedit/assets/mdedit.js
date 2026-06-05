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
 *   opts = { livePreview: bool, toolbar: "minimal"|"standard"|"full",
 *            uploadURL: string }  // uploadURL "" disables image upload
 *   controller = {
 *     getValue(): string        // current editor content
 *     setValue(text): void      // replace editor content (used by file load)
 *     sync(): void              // flush editor content into textarea.value
 *     destroy(): void           // tear down, restore the bare textarea
 *     onChange(cb): void        // optional; cb() on each edit (for live preview)
 *     focus(): void             // optional; put the caret in the editor
 *   }
 *
 * Responsibilities the loader owns (so adapters stay thin):
 *   - enhancing [data-mdedit] textareas on load and after htmx swaps
 *   - flushing every editor's value into its textarea before any htmx
 *     request, so Save/Preview serialize current content
 *   - flushing editors on a plain (non-htmx) <form> submit too, and taking
 *     over `required` for hidden textareas — the editor hides the real
 *     textarea, so native `required` would silently block submit on an
 *     unfocusable control; the loader re-enforces it with a visible message
 *   - tearing down editors when htmx removes their element
 *   - wiring debounced server-side live preview when data-mdedit-live is set
 *   - wiring [data-mdedit-load] file inputs: read a local Markdown file in
 *     the browser and drop it into the editor (Field.AllowFileLoad). The file
 *     is never uploaded — it becomes the textarea value and rides the normal
 *     Save POST, so it adds no server attack surface.
 */
(function () {
  "use strict";

  var registry = Object.create(null);
  var DEFAULT_ADAPTER = "easymde";

  // Cap for client-side file loads. Markdown is text; this only stops someone
  // dropping a huge binary into the DOM. The server still enforces its own
  // limits on Save regardless of this.
  var MAX_LOAD_BYTES = 2 * 1024 * 1024;

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
    wireFileLoaders(scope);
    if (root && root.matches && root.matches("input[data-mdedit-load]")) {
      wireFileInput(root);
    }
  }

  // Wire every [data-mdedit-load] file input in scope. The input's value names
  // the id of the textarea it loads into. The editor controller is resolved at
  // change time (ta._mdeditController), so wiring order vs. mountOne does not
  // matter and re-enhancement after an htmx swap stays correct.
  function wireFileLoaders(scope) {
    if (!scope.querySelectorAll) return;
    var inputs = scope.querySelectorAll("input[data-mdedit-load]");
    for (var i = 0; i < inputs.length; i++) {
      wireFileInput(inputs[i]);
    }
  }

  function wireFileInput(input) {
    if (input._mdeditLoadWired) return; // idempotent across re-enhancement
    input._mdeditLoadWired = true;
    input.addEventListener("change", function () {
      var ta = document.getElementById(input.getAttribute("data-mdedit-load"));
      var file = input.files && input.files[0];
      if (!ta || !file) return;
      if (file.size > MAX_LOAD_BYTES) {
        window.alert(
          "That file is too large to load (max " +
            Math.round(MAX_LOAD_BYTES / 1024) +
            " KB)."
        );
        input.value = "";
        return;
      }
      var reader = new FileReader();
      reader.onload = function () {
        var text = reader.result == null ? "" : String(reader.result);
        // A NUL byte means this is not a text file; refuse rather than paste
        // garbage into the editor.
        if (text.indexOf("\u0000") !== -1) {
          window.alert("That does not look like a text file.");
          input.value = "";
          return;
        }
        var controller = ta._mdeditController;
        var current = controller ? controller.getValue() : ta.value;
        // Loading replaces the buffer; confirm before discarding real content.
        if (
          current &&
          current.length &&
          !window.confirm("Replace the current content with the uploaded file?")
        ) {
          input.value = "";
          return;
        }
        if (controller && typeof controller.setValue === "function") {
          controller.setValue(text);
        } else {
          ta.value = text; // degraded: no adapter mounted, plain textarea
        }
        input.value = ""; // let the author re-load the same file
      };
      reader.onerror = function () {
        window.alert("Could not read that file.");
        input.value = "";
      };
      reader.readAsText(file);
    });
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
    var uploadURL = ta.getAttribute("data-mdedit-upload") || "";
    var controller = mount(ta, {
      livePreview: live,
      toolbar: toolbar,
      uploadURL: uploadURL,
    });
    ta._mdeditController = controller;

    // The adapter has now replaced the textarea with its own editor surface and
    // hidden the real <textarea>. A hidden control can't satisfy or display
    // native constraint validation: an empty `required` textarea makes the
    // browser refuse to submit with the unfocusable "invalid form control"
    // error — silently, no request. So take over the requirement. Record it,
    // strip the native attribute from the now-hidden field, and re-enforce it
    // ourselves in the form's submit guard, where we can flush the editor
    // first and surface a visible message. (If enhancement had failed above we
    // return early and leave the visible textarea's native `required` intact.)
    if (ta.required) {
      ta.dataset.mdeditRequired = "1";
      ta.required = false;
    }
    if (ta.form) guardForm(ta.form);

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

  // Plain (non-htmx) <form> submits never fire htmx:configRequest, so they
  // would serialize without the configRequest flush above. guardForm closes
  // that gap: it installs one capturing submit listener per form that flushes
  // every mdedit editor in the form before the browser serializes it, and
  // re-enforces the requirement we lifted off the hidden textareas in mountOne.
  // Capture phase so we run before htmx's own bubble-phase submit handling.
  function guardForm(form) {
    if (form._mdeditGuard) return; // idempotent across re-enhancement
    form._mdeditGuard = true;
    form.addEventListener(
      "submit",
      function (e) {
        var nodes = form.querySelectorAll("textarea[data-mdedit]");
        var blocker = null;
        for (var i = 0; i < nodes.length; i++) {
          var ta = nodes[i];
          if (ta._mdeditController) ta._mdeditController.sync();
          clearError(ta);
          if (ta.dataset.mdeditRequired === "1" && !ta.value.trim()) {
            showError(ta, "This field is required.");
            if (!blocker) blocker = ta;
          }
        }
        if (blocker) {
          e.preventDefault();
          if (blocker._mdeditController && blocker._mdeditController.focus) {
            blocker._mdeditController.focus();
          }
          var wrap = blocker.closest ? blocker.closest(".mdedit") : null;
          if (wrap && wrap.scrollIntoView) {
            wrap.scrollIntoView({ block: "center" });
          }
        }
      },
      true
    );
  }

  // The required-field message lives in a module-owned node inside the editor
  // wrapper (.mdedit), so it needs no host CSS and survives htmx swaps with the
  // editor. clearError hides it; showError fills and reveals it.
  function errorNode(ta) {
    var wrap = ta.closest ? ta.closest(".mdedit") : null;
    if (!wrap) return null;
    var node = wrap.querySelector(".mdedit-error");
    if (!node) {
      node = document.createElement("p");
      node.className = "mdedit-error";
      node.setAttribute("role", "alert");
      node.hidden = true;
      wrap.appendChild(node);
    }
    return node;
  }
  function showError(ta, msg) {
    var node = errorNode(ta);
    if (!node) return;
    node.textContent = msg;
    node.hidden = false;
  }
  function clearError(ta) {
    var wrap = ta.closest ? ta.closest(".mdedit") : null;
    var node = wrap && wrap.querySelector(".mdedit-error");
    if (node) {
      node.textContent = "";
      node.hidden = true;
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
