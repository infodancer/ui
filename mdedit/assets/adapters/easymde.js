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
 *     font from a CDN — the module stays self-contained. We supply a
 *     compact toolbar with text labels instead. (Vendoring an SVG icon set
 *     for a prettier toolbar is a follow-up; see README.)
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

  // Text-labelled toolbar buttons: friendly for non-technical authors, no
  // icon font required. Defined once and composed into profiles below.
  var btn = {
    bold: { name: "bold", action: EasyMDE.toggleBold, text: "B", title: "Bold" },
    italic: { name: "italic", action: EasyMDE.toggleItalic, text: "I", title: "Italic" },
    strikethrough: { name: "strikethrough", action: EasyMDE.toggleStrikethrough, text: "S̶", title: "Strikethrough" },
    heading: { name: "heading", action: EasyMDE.toggleHeadingSmaller, text: "H", title: "Heading" },
    quote: { name: "quote", action: EasyMDE.toggleBlockquote, text: "❝", title: "Quote" },
    ul: { name: "unordered-list", action: EasyMDE.toggleUnorderedList, text: "•", title: "Bulleted list" },
    ol: { name: "ordered-list", action: EasyMDE.toggleOrderedList, text: "1.", title: "Numbered list" },
    link: { name: "link", action: EasyMDE.drawLink, text: "🔗", title: "Insert link" },
    code: { name: "code", action: EasyMDE.toggleCodeBlock, text: "</>", title: "Code block" },
  };

  // Profiles match Field.Toolbar. They shape what the editor offers; the
  // server's markdown preset is what actually constrains the output, so the
  // two should agree (e.g. "standard" alongside markdown.Comment).
  var profiles = {
    minimal: [btn.bold, btn.italic, "|", btn.link],
    standard: [btn.bold, btn.italic, btn.strikethrough, "|", btn.quote, btn.ul, btn.ol, "|", btn.link, btn.code],
    full: [btn.bold, btn.italic, btn.heading, "|", btn.quote, btn.ul, btn.ol, "|", btn.link, btn.code],
  };

  expose("easymde", function (textarea, opts) {
    var editor = new EasyMDE({
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
    });

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
