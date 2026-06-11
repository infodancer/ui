// infodancer/ui — image lightbox.
//
// A tiny, dependency-free viewer on the native <dialog> element, declarative
// like track.js: ui owns the mechanism, the consumer owns the markup —
//
//   <a href="photo-full.jpg" data-lightbox="trip"><img src="thumb.jpg" alt="A caption"></a>
//
// Links sharing a data-lightbox value form a gallery in document order:
// prev/next buttons and arrow keys navigate it, and a counter shows the
// position. The caption comes from data-lightbox-caption, else the linked
// image's alt text. showModal() supplies ESC-to-close, focus containment, and
// the ::backdrop for free; clicking outside the figure also closes. Without
// JavaScript the links simply open the full image — the lightbox is
// progressive enhancement, so it never breaks a page when blocked.
//
// One delegated click listener covers the whole document, so galleries added
// after load (htmx swaps, infinite scroll) need no re-initialization.
(function () {
  var dialog = null;
  var img, caption, counter, prevBtn, nextBtn;
  var items = [];
  var index = 0;

  function build() {
    dialog = document.createElement('dialog');
    dialog.className = 'app-lightbox';
    dialog.setAttribute('aria-label', 'Image viewer');

    var figure = document.createElement('figure');
    img = document.createElement('img');
    caption = document.createElement('figcaption');
    figure.appendChild(img);
    figure.appendChild(caption);

    // Glyphs by escape so the source stays ASCII: U+00D7 multiplication sign
    // (close), U+2039/U+203A single angle quotes (prev/next).
    var close = button('app-lightbox-close', 'Close', '\u00D7', function () {
      dialog.close();
    });
    prevBtn = button('app-lightbox-prev', 'Previous image', '\u2039', function () {
      show(index - 1);
    });
    nextBtn = button('app-lightbox-next', 'Next image', '\u203A', function () {
      show(index + 1);
    });
    counter = document.createElement('span');
    counter.className = 'app-lightbox-counter';

    dialog.appendChild(close);
    dialog.appendChild(prevBtn);
    dialog.appendChild(figure);
    dialog.appendChild(nextBtn);
    dialog.appendChild(counter);

    // A click that lands on the dialog itself (not its children) is a click
    // on the backdrop area: light-dismiss.
    dialog.addEventListener('click', function (e) {
      if (e.target === dialog) dialog.close();
    });

    dialog.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowLeft') show(index - 1);
      else if (e.key === 'ArrowRight') show(index + 1);
    });

    document.body.appendChild(dialog);
  }

  function button(className, label, glyph, onClick) {
    var b = document.createElement('button');
    b.type = 'button';
    b.className = className;
    b.setAttribute('aria-label', label);
    b.textContent = glyph;
    b.addEventListener('click', onClick);
    return b;
  }

  function captionFor(a) {
    var explicit = a.getAttribute('data-lightbox-caption');
    if (explicit) return explicit;
    var thumb = a.querySelector('img');
    return (thumb && thumb.alt) || '';
  }

  function show(i) {
    index = (i + items.length) % items.length;
    var a = items[index];
    img.src = a.href;
    img.alt = captionFor(a);
    caption.textContent = captionFor(a);
    var multi = items.length > 1;
    prevBtn.hidden = !multi;
    nextBtn.hidden = !multi;
    counter.hidden = !multi;
    counter.textContent = multi ? (index + 1) + ' / ' + items.length : '';
  }

  document.addEventListener('click', function (e) {
    if (!e.target.closest) return;
    var a = e.target.closest('a[data-lightbox]');
    if (!a || !a.href) return;
    // Defer to deliberate new-tab/window clicks.
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
    e.preventDefault();

    // Collect the gallery fresh on every open: matches by attribute equality
    // in JS rather than a selector, so group names need no CSS escaping, and
    // content swapped in after load is picked up automatically.
    var group = a.getAttribute('data-lightbox');
    var all = document.querySelectorAll('a[data-lightbox]');
    items = [];
    for (var i = 0; i < all.length; i++) {
      if (all[i].getAttribute('data-lightbox') === group) items.push(all[i]);
    }
    if (!items.length) return;

    if (!dialog) build();
    show(items.indexOf(a));
    dialog.showModal();
  });
})();
