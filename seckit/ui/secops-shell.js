(function (global) {
  'use strict';

  var THEME_KEY = 'secops_theme';

  function isValidTheme(val) {
    return val === 'dark' || val === 'light';
  }

  function applyTheme(theme) {
    var root = document.documentElement;
    if (isValidTheme(theme)) {
      root.dataset.theme = theme;
      root.classList.toggle('dark', theme === 'dark');
    } else {
      delete root.dataset.theme;
      var prefersDark = Boolean(global.matchMedia && global.matchMedia('(prefers-color-scheme: dark)').matches);
      root.classList.toggle('dark', prefersDark);
    }
  }

  function effectiveTheme() {
    var explicit = document.documentElement.dataset.theme;
    if (isValidTheme(explicit)) return explicit;
    return global.matchMedia && global.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  function saveTheme(theme) {
    try {
      if (isValidTheme(theme)) {
        global.localStorage.setItem(THEME_KEY, theme);
      }
    } catch (_) {
      // Ignore storage restriction.
    }
  }

  function initTheme(legacyKeys) {
    var saved = '';
    try {
      saved = global.localStorage.getItem(THEME_KEY) || '';
      if (!isValidTheme(saved) && Array.isArray(legacyKeys)) {
        for (var i = 0; i < legacyKeys.length; i++) {
          var legacy = global.localStorage.getItem(legacyKeys[i]);
          if (isValidTheme(legacy)) {
            saved = legacy;
            saveTheme(saved);
            break;
          }
        }
      }
    } catch (_) {
      // Ignore storage restriction.
    }
    applyTheme(isValidTheme(saved) ? saved : '');
    return effectiveTheme();
  }

  function toggleTheme() {
    var next = effectiveTheme() === 'dark' ? 'light' : 'dark';
    applyTheme(next);
    saveTheme(next);
    try {
      global.dispatchEvent(new CustomEvent('secops-theme-change', { detail: { theme: next } }));
    } catch (_) {
      // Ignore Event error in constrained environments.
    }
    return next;
  }

  function toast(message, type, duration) {
    if (!message || !document.body) return;
    var stack = document.getElementById('secops-toast-stack');
    if (!stack) {
      stack = document.createElement('div');
      stack.id = 'secops-toast-stack';
      stack.className = 'secops-toast-stack';
      stack.setAttribute('aria-live', 'polite');
      document.body.appendChild(stack);
    }
    var kind = type === 'success' || type === 'error' || type === 'warning' ? type : 'info';
    var item = document.createElement('div');
    item.className = 'secops-toast is-' + kind;
    item.setAttribute('role', kind === 'error' ? 'alert' : 'status');
    item.textContent = String(message);
    stack.appendChild(item);

    var ms = typeof duration === 'number' && duration > 0 ? duration : 3200;
    global.setTimeout(function () {
      item.remove();
    }, ms);
  }

  function copyIconSVG() {
    var ns = 'http://www.w3.org/2000/svg';
    var svg = document.createElementNS(ns, 'svg');
    svg.setAttribute('viewBox', '0 0 16 16');
    svg.setAttribute('fill', 'none');
    svg.setAttribute('aria-hidden', 'true');
    var rect = document.createElementNS(ns, 'rect');
    rect.setAttribute('x', '5.5');
    rect.setAttribute('y', '5.5');
    rect.setAttribute('width', '7.5');
    rect.setAttribute('height', '7.5');
    rect.setAttribute('rx', '1.5');
    rect.setAttribute('stroke', 'currentColor');
    rect.setAttribute('stroke-width', '1.3');
    var path = document.createElementNS(ns, 'path');
    path.setAttribute('d', 'M10.5 3.5H4.25A1.25 1.25 0 0 0 3 4.75V11');
    path.setAttribute('stroke', 'currentColor');
    path.setAttribute('stroke-width', '1.3');
    path.setAttribute('stroke-linecap', 'round');
    svg.appendChild(rect);
    svg.appendChild(path);
    return svg;
  }

  function fallbackCopy(value) {
    var field = document.createElement('textarea');
    field.value = value;
    field.setAttribute('readonly', '');
    field.style.position = 'fixed';
    field.style.left = '-9999px';
    field.style.top = '0';
    field.style.opacity = '0';
    document.body.appendChild(field);
    field.select();
    var ok = false;
    try {
      ok = document.execCommand('copy');
    } catch (_) {
      ok = false;
    } finally {
      field.remove();
    }
    return ok;
  }

  function copy(value, button, label) {
    var str = String(value == null ? '' : value);
    if (!str) return Promise.resolve(false);

    var onSuccess = function () {
      if (button && button.nodeType === 1) {
        if (!button._secopsPrevChildren) {
          button._secopsPrevChildren = Array.prototype.slice.call(button.childNodes);
        }
        button.classList.add('is-copied');
        button.textContent = '已复制';
        global.clearTimeout(button._secopsCopyTimer);
        button._secopsCopyTimer = global.setTimeout(function () {
          button.classList.remove('is-copied');
          if (button._secopsPrevChildren && button._secopsPrevChildren.length) {
            button.replaceChildren.apply(button, button._secopsPrevChildren);
          } else {
            button.replaceChildren(copyIconSVG());
          }
        }, 1500);
      }
      toast('已复制 ' + (label || str), 'success', 2000);
      return true;
    };

    var onFail = function () {
      toast('复制失败，请手动选择复制', 'error');
      return false;
    };

    if (global.navigator && global.navigator.clipboard && global.navigator.clipboard.writeText) {
      return global.navigator.clipboard.writeText(str).then(onSuccess).catch(function () {
        return fallbackCopy(str) ? onSuccess() : onFail();
      });
    }
    return Promise.resolve(fallbackCopy(str) ? onSuccess() : onFail());
  }

  function cveUrl(id) {
    return 'https://www.cve.org/CVERecord?id=' + encodeURIComponent(String(id || '').trim().toUpperCase());
  }

  function createCveNode(id) {
    var cve = String(id || '').trim().toUpperCase();
    var wrap = document.createElement('div');
    wrap.className = 'secops-cve-cell';
    var a = document.createElement('a');
    a.className = 'secops-cve-link';
    a.href = cveUrl(cve);
    a.target = '_blank';
    a.rel = 'noopener noreferrer';
    a.title = '在 cve.org 查看 ' + cve;
    a.textContent = cve + ' ↗';
    var btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'secops-copy-btn';
    btn.setAttribute('aria-label', '复制 ' + cve);
    btn.title = '复制 ' + cve;
    btn.appendChild(copyIconSVG());
    btn.addEventListener('click', function (e) {
      e.stopPropagation();
      copy(cve, btn, cve);
    });
    wrap.appendChild(a);
    wrap.appendChild(btn);
    return wrap;
  }

  function isSafeNavUrl(raw) {
    if (typeof raw !== 'string') return false;
    var val = raw.trim();
    if (!val) return false;
    if (val.charAt(0) === '/' && val.charAt(1) !== '/') return true;
    return /^https?:\/\/\S+$/i.test(val);
  }

  function mountNav(container, options) {
    if (!container) return Promise.resolve();
    var endpoint = (options && options.endpoint) || '/secops/nav.json';
    return global.fetch(endpoint, { credentials: 'same-origin', cache: 'no-store' })
      .then(function (resp) {
        if (!resp.ok) throw new Error('HTTP ' + resp.status);
        return resp.json();
      })
      .then(function (data) {
        var items = Array.isArray(data && data.items) ? data.items : [];
        var current = (data && data.current) || '';
        var frag = document.createDocumentFragment();
        for (var i = 0; i < items.length; i++) {
          var item = items[i];
          if (!item || !item.label || !isSafeNavUrl(item.url)) continue;
          var a = document.createElement('a');
          a.className = 'secops-nav-item';
          a.href = item.url;
          a.textContent = item.label;
          if (current && item.key === current) {
            a.classList.add('is-active');
            a.setAttribute('aria-current', 'page');
          } else {
            a.target = '_self';
          }
          frag.appendChild(a);
        }
        container.replaceChildren(frag);
      })
      .catch(function () {
        // Leave empty when /secops/nav.json is unavailable.
      });
  }

  function isEditableElement(el) {
    if (!el || el.nodeType !== 1) return false;
    var tag = el.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || el.isContentEditable) return true;
    var role = el.getAttribute && el.getAttribute('role');
    return role === 'combobox' || role === 'option' || role === 'textbox';
  }

  function resolveTarget(getter) {
    if (typeof getter === 'function') return getter();
    if (typeof getter === 'string') return document.querySelector(getter);
    return getter || null;
  }

  function bindShortcuts(options) {
    var opts = options || {};
    var handler = function (event) {
      if (event.defaultPrevented) return;
      if (event.key === '/' && !event.ctrlKey && !event.metaKey && !event.altKey) {
        if (isEditableElement(document.activeElement)) return;
        var input = resolveTarget(opts.getSearchInput);
        if (input && typeof input.focus === 'function') {
          event.preventDefault();
          input.focus();
          if (typeof input.select === 'function') input.select();
        }
        return;
      }
      if (event.key === 'Escape') {
        var openDialog = document.querySelector('dialog[open]');
        if (openDialog && typeof openDialog.close === 'function') {
          openDialog.close();
          return;
        }
        var openOverlay = document.querySelector('.overlay.open');
        if (openOverlay) {
          openOverlay.classList.remove('open');
          return;
        }
        if (typeof opts.onEscape === 'function' && opts.onEscape(event) === true) {
          return;
        }
        var activeInput = resolveTarget(opts.getSearchInput);
        if (activeInput && document.activeElement === activeInput) {
          if (activeInput.value) {
            if (typeof opts.onClear === 'function') {
              opts.onClear();
            } else {
              activeInput.value = '';
              activeInput.dispatchEvent(new Event('input', { bubbles: true }));
            }
          } else {
            activeInput.blur();
          }
        }
      }
    };
    document.addEventListener('keydown', handler);
    return function () {
      document.removeEventListener('keydown', handler);
    };
  }

  function enhanceSearchInput(inputOrSelector, options) {
    var input = resolveTarget(inputOrSelector);
    if (!input) return function () {};
    var opts = options || {};
    var clearBtn = resolveTarget(opts.clearButton);
    if (!clearBtn && input.parentElement) {
      clearBtn = input.parentElement.querySelector('[data-secops-search-clear], .secops-search-clear');
    }
    var sync = function () {
      if (clearBtn) clearBtn.hidden = !String(input.value || '').trim();
    };
    var onInput = function () {
      sync();
    };
    var onClearClick = function () {
      input.value = '';
      sync();
      if (typeof opts.onClear === 'function') {
        opts.onClear();
      } else {
        input.dispatchEvent(new Event('input', { bubbles: true }));
      }
      if (typeof input.focus === 'function') input.focus();
    };
    input.addEventListener('input', onInput);
    if (clearBtn) clearBtn.addEventListener('click', onClearClick);
    sync();
    return function () {
      input.removeEventListener('input', onInput);
      if (clearBtn) clearBtn.removeEventListener('click', onClearClick);
    };
  }

  function initThemeToggles() {
    var buttons = document.querySelectorAll('[data-secops-theme-toggle]');
    for (var i = 0; i < buttons.length; i++) {
      (function (btn) {
        if (btn.dataset.secopsThemeBound) return;
        btn.dataset.secopsThemeBound = '1';
        btn.addEventListener('click', function () {
          toggleTheme();
        });
      })(buttons[i]);
    }
  }

  function mountAllNavs() {
    var navs = document.querySelectorAll('[data-secops-nav]');
    for (var i = 0; i < navs.length; i++) {
      mountNav(navs[i], { endpoint: navs[i].getAttribute('data-secops-nav') || '/secops/nav.json' });
    }
  }

  // Apply saved theme immediately to avoid flash of unstyled light/dark state.
  initTheme();

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      initThemeToggles();
      mountAllNavs();
    });
  } else {
    initThemeToggles();
    mountAllNavs();
  }

  global.SecOps = {
    THEME_KEY: THEME_KEY,
    initTheme: initTheme,
    effectiveTheme: effectiveTheme,
    toggleTheme: toggleTheme,
    toast: toast,
    copy: copy,
    copyIconSVG: copyIconSVG,
    cveUrl: cveUrl,
    createCveNode: createCveNode,
    mountNav: mountNav,
    bindShortcuts: bindShortcuts,
    enhanceSearchInput: enhanceSearchInput
  };
})(window);
