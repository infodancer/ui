// infodancer/ui — action tracker.
//
// A tiny, framework-agnostic companion to the ui/analytics partial. That
// partial emits the analytics vendor's own <script>; this turns declarative
// data-track attributes into the vendor's custom events. ui owns the mechanism;
// the consumer owns the vocabulary — which elements carry data-track and what
// the event names and categories mean.
//
//   window.uiTrack(event, data) — fire a custom event, best-effort.
//
//   data-track="event_name" on a <form> (fires on submit) or any other
//   clickable element (fires on click). data-track-* attributes become event
//   properties: data-track-category="ai" data-track-tool="encounter" →
//   {category: "ai", tool: "encounter"}.
//
// One delegated submit listener and one delegated click listener cover the
// whole document, so a new action costs an HTML attribute, not JavaScript.
// All dispatch is null-guarded: the analytics script loads async and may be
// blocked, and analytics must never break a user action.
(function () {
  function dispatch(event, data) {
    if (window.umami && typeof window.umami.track === 'function') {
      window.umami.track(event, data);
    }
    // Plausible arm slots in here when a consumer needs it:
    //   if (window.plausible) window.plausible(event, { props: data });
  }

  window.uiTrack = function (event, data) {
    try {
      if (event) dispatch(event, data || {});
    } catch (e) { /* analytics is non-essential; swallow */ }
  };

  // Build the event-properties object from an element's data-track-* attributes.
  function propsFrom(el) {
    var data = {};
    var attrs = el.attributes;
    var prefix = 'data-track-';
    for (var i = 0; i < attrs.length; i++) {
      var name = attrs[i].name;
      if (name.indexOf(prefix) === 0) {
        data[name.slice(prefix.length).replace(/-/g, '_')] = attrs[i].value;
      }
    }
    return data;
  }

  function fire(el) {
    window.uiTrack(el.getAttribute('data-track'), propsFrom(el));
  }

  // Forms track on submit (capture phase, so it runs even when a page script
  // calls preventDefault — e.g. an AJAX form). Put data-track on the <form>.
  document.addEventListener('submit', function (e) {
    var form = e.target.closest && e.target.closest('form[data-track]');
    if (form) fire(form);
  }, true);

  // Everything else tracks on click. Form submit buttons are handled by the
  // submit listener above; skip them here to avoid double counting.
  document.addEventListener('click', function (e) {
    if (!e.target.closest) return;
    var el = e.target.closest('[data-track]');
    if (!el || el.tagName === 'FORM') return;
    if (el.type === 'submit' && el.form) return;
    fire(el);
  }, true);
})();
