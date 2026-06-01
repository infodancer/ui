/*
 * EasyMDE adapter for the mdedit seam.
 *
 * Two deliberate constraints, both for the security/correctness story:
 *
 *  1. EasyMDE's built-in preview (its bundled marked.js) is NOT enabled.
 *     The toolbar omits "preview", "side-by-side", and "fullscreen", so
 *     the only renderer is the server's goldmark + bluemonday. marked.js
 *     would disagree with goldmark and show the author a preview that
 *     differs from what gets stored.
 *
 *  2. autoDownloadFontAwesome is false so EasyMDE never fetches an icon
 *     font from a CDN — the module stays self-contained. Buttons render an
 *     inline SVG via each item's `icon` (which EasyMDE drops in as the
 *     button's innerHTML); no icon font, no network. The glyphs are from
 *     Tabler Icons (MIT) — see assets/vendor/PROVENANCE.md for attribution.
 */
(function () {
  "use strict";
  if (typeof EasyMDE === "undefined") {
    console.error("mdedit: easymde.min.js must load before the EasyMDE adapter");
    return;
  }
  // Register against the loader if it is ready, else queue for it to drain.
  // This makes loader/adapter load order irrelevant.
  var md = (window.mdedit = window.mdedit || {});
  function expose(name, mount) {
    if (typeof md.register === "function") md.register(name, mount);
    else (md._pending = md._pending || []).push([name, mount]);
  }

  // Wrap a set of inline path data in a Tabler-style 24×24 stroke SVG.
  // stroke="currentColor" so the icon picks up the toolbar button color
  // (set from --app-color-fg in mdedit.css); CSS sizes it. aria-hidden
  // because the button already carries an aria-label from `title`.
  function svg(paths) {
    return (
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" ' +
      'fill="none" stroke="currentColor" stroke-width="2" ' +
      'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' +
      paths +
      "</svg>"
    );
  }

  // Tabler Icons (MIT) path data, by button.
  var icons = {
    bold: '<path d="M7 5h6a3.5 3.5 0 0 1 0 7h-6l0 -7" /><path d="M13 12h1a3.5 3.5 0 0 1 0 7h-7v-7" />',
    italic: '<path d="M11 5l6 0" /><path d="M7 19l6 0" /><path d="M14 5l-4 14" />',
    strikethrough: '<path d="M5 12l14 0" /><path d="M16 6.5a4 2 0 0 0 -4 -1.5h-1a3.5 3.5 0 0 0 0 7h2a3.5 3.5 0 0 1 0 7h-1.5a4 2 0 0 1 -4 -1.5" />',
    heading: '<path d="M7 12h10" /><path d="M7 5v14" /><path d="M17 5v14" />',
    quote: '<path d="M6 15h15" /><path d="M21 19h-15" /><path d="M15 11h6" /><path d="M21 7h-6" /><path d="M9 9h1a1 1 0 1 1 -1 1v-2.5a2 2 0 0 1 2 -2" /><path d="M3 9h1a1 1 0 1 1 -1 1v-2.5a2 2 0 0 1 2 -2" />',
    ul: '<path d="M9 6l11 0" /><path d="M9 12l11 0" /><path d="M9 18l11 0" /><path d="M5 6l0 .01" /><path d="M5 12l0 .01" /><path d="M5 18l0 .01" />',
    ol: '<path d="M11 6h9" /><path d="M11 12h9" /><path d="M12 18h8" /><path d="M4 16a2 2 0 1 1 4 0c0 .591 -.5 1 -1 1.5l-3 2.5h4" /><path d="M6 10v-6l-2 2" />',
    link: '<path d="M9 15l6 -6" /><path d="M11 6l.463 -.536a5 5 0 0 1 7.071 7.072l-.534 .464" /><path d="M13 18l-.397 .534a5.068 5.068 0 0 1 -7.127 0a4.972 4.972 0 0 1 0 -7.071l.524 -.463" />',
    code: '<path d="M7 8l-4 4l4 4" /><path d="M17 8l4 4l-4 4" /><path d="M14 4l-4 16" />',
    image: '<path d="M15 8h.01" /><path d="M3 6a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v12a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3v-12" /><path d="M3 16l5 -5c.928 -.893 2.072 -.893 3 0l5 5" /><path d="M14 14l1 -1c.928 -.893 2.072 -.893 3 0l3 3" />',
    table: '<path d="M3 5a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v14a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2v-14" /><path d="M3 10h18" /><path d="M10 3v18" />',
    hr: '<path d="M5 12h2" /><path d="M11 12h2" /><path d="M17 12h2" />',
  };

  // Toolbar buttons, defined once and composed into profiles below. Each
  // carries an inline SVG icon (no icon font) and an action from EasyMDE.
  var btn = {
    bold: { name: "bold", action: EasyMDE.toggleBold, icon: svg(icons.bold), title: "Bold" },
    italic: { name: "italic", action: EasyMDE.toggleItalic, icon: svg(icons.italic), title: "Italic" },
    strikethrough: { name: "strikethrough", action: EasyMDE.toggleStrikethrough, icon: svg(icons.strikethrough), title: "Strikethrough" },
    heading: { name: "heading", action: EasyMDE.toggleHeadingSmaller, icon: svg(icons.heading), title: "Heading" },
    quote: { name: "quote", action: EasyMDE.toggleBlockquote, icon: svg(icons.quote), title: "Quote" },
    ul: { name: "unordered-list", action: EasyMDE.toggleUnorderedList, icon: svg(icons.ul), title: "Bulleted list" },
    ol: { name: "ordered-list", action: EasyMDE.toggleOrderedList, icon: svg(icons.ol), title: "Numbered list" },
    link: { name: "link", action: EasyMDE.drawLink, icon: svg(icons.link), title: "Insert link" },
    code: { name: "code", action: EasyMDE.toggleCodeBlock, icon: svg(icons.code), title: "Code block" },
    image: { name: "image", action: EasyMDE.drawImage, icon: svg(icons.image), title: "Insert image" },
    table: { name: "table", action: EasyMDE.drawTable, icon: svg(icons.table), title: "Insert table" },
    hr: { name: "horizontal-rule", action: EasyMDE.drawHorizontalRule, icon: svg(icons.hr), title: "Horizontal rule" },
  };

  // Profiles match Field.Toolbar. They shape what the editor offers; the
  // server's markdown preset is what actually constrains the output, so the
  // two should agree (e.g. "full" alongside a renderer with tables/images on).
  var profiles = {
    minimal: [btn.bold, btn.italic, "|", btn.link],
    standard: [btn.bold, btn.italic, btn.strikethrough, btn.heading, "|", btn.quote, btn.ul, btn.ol, "|", btn.link, btn.code],
    full: [btn.bold, btn.italic, btn.strikethrough, btn.heading, "|", btn.quote, btn.ul, btn.ol, "|", btn.link, btn.code, btn.image, "|", btn.table, btn.hr],
  };

  // Upload an image to the host endpoint and hand EasyMDE back the URL it
  // should insert. EasyMDE calls this for drag-drop, paste, and the browse
  // dialog when uploadImage is on. The endpoint owns auth, validation, and
  // re-encoding (see Field.UploadURL); we only POST multipart "image",
  // same-origin, and read {url} / {error} from the JSON reply.
  function makeUploadFn(url) {
    return function (file, onSuccess, onError) {
      var form = new FormData();
      form.append("image", file);
      fetch(url, {
        method: "POST",
        body: form,
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      })
        .then(function (resp) {
          return resp
            .json()
            .catch(function () {
              return {};
            })
            .then(function (body) {
              if (resp.ok && body && body.url) onSuccess(body.url);
              else onError((body && body.error) || "Image upload failed.");
            });
        })
        .catch(function () {
          onError("Image upload failed.");
        });
    };
  }

  expose("easymde", function (textarea, opts) {
    var config = {
      element: textarea,
      autoDownloadFontAwesome: false,
      toolbar: profiles[opts.toolbar] || profiles.full,
      status: false,
      spellChecker: false,
      // Belt and suspenders: even if a preview button slipped in, point it
      // at nothing useful — the server owns rendering.
      previewRender: function () {
        return "Preview is rendered by the server.";
      },
    };
    // Inline image upload (drag-drop / paste / browse) when the host wired an
    // endpoint. uploadImage turns the feature on; imageUploadFunction routes
    // the bytes to that endpoint instead of EasyMDE's own URL-based default.
    if (opts.uploadURL) {
      config.uploadImage = true;
      config.imageUploadFunction = makeUploadFn(opts.uploadURL);
    }
    var editor = new EasyMDE(config);

    return {
      getValue: function () {
        return editor.value();
      },
      setValue: function (text) {
        // Replaces the buffer (file load). EasyMDE mirrors this back into the
        // underlying textarea, keeping it authoritative for Save/Preview.
        editor.value(text);
      },
      sync: function () {
        // EasyMDE keeps the original textarea updated, but flush explicitly
        // so a request never races a pending CodeMirror change.
        textarea.value = editor.value();
      },
      destroy: function () {
        // Restores the original <textarea> so htmx can cleanly remove it.
        editor.toTextArea();
      },
      onChange: function (cb) {
        editor.codemirror.on("change", cb);
      },
    };
  });
})();
