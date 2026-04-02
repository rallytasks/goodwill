// Goodwill Feedback Widget (FAB)
// Compact modal with Enter to submit, Shift+Enter for newline.
(function () {
  'use strict';

  var WIDGET_ID = 'goodwill-feedback-widget';
  if (document.getElementById(WIDGET_ID)) return;

  var isOpen = false;
  var canSubmit = false;

  function checkAuth() {
    fetch('/api/donor/profile', { credentials: 'same-origin' })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) {
        if (data && data.can_submit_feedback && !canSubmit) {
          canSubmit = true;
          init();
        }
      })
      .catch(function () {});
  }
  checkAuth();

  var statusLabels = {
    'new': 'Submitted',
    'reviewed': 'Under Review',
    'roadmapped': 'In Progress',
    'done': 'Shipped'
  };

  function injectStyles() {
    var css = document.createElement('style');
    css.textContent = `
      #gw-fb-fab {
        position: fixed; bottom: 24px; right: 24px; z-index: 99999;
        width: 48px; height: 48px; border-radius: 50%;
        background: #004B87; color: #fff; border: none; cursor: pointer;
        box-shadow: 0 4px 14px rgba(0,75,135,0.4);
        display: flex; align-items: center; justify-content: center;
        transition: transform 0.15s ease;
      }
      #gw-fb-fab:hover { transform: scale(1.08); }
      #gw-fb-fab:active { transform: scale(0.95); }
      #gw-fb-fab svg { width: 22px; height: 22px; fill: currentColor; }

      #gw-fb-backdrop {
        position: fixed; inset: 0; z-index: 100000;
        background: rgba(0,0,0,0.4); display: none;
      }
      #gw-fb-backdrop.open { display: block; }

      #gw-fb-modal {
        position: fixed; z-index: 100001; display: none;
        bottom: 88px; right: 24px;
        width: 380px; max-width: calc(100vw - 32px);
        max-height: 70vh;
        background: #fff; border-radius: 16px;
        box-shadow: 0 8px 40px rgba(0,0,0,0.2);
        flex-direction: column; overflow: hidden;
      }
      #gw-fb-modal.open { display: flex; }

      @media (max-width: 480px) {
        #gw-fb-modal {
          bottom: 0; right: 0; left: 0;
          width: 100%; max-width: 100%;
          border-radius: 16px 16px 0 0;
          max-height: 85vh;
        }
      }

      .gw-fb-header {
        display: flex; align-items: center; padding: 12px 16px;
        border-bottom: 1px solid #e5e7eb; gap: 8px; flex-shrink: 0;
      }
      .gw-fb-header h2 { margin: 0; font-size: 15px; font-weight: 600; color: #1a1a1a; flex: 1; }
      .gw-fb-close {
        background: none; border: none; cursor: pointer; font-size: 20px;
        color: #999; padding: 2px 6px; border-radius: 6px; line-height: 1; flex-shrink: 0;
      }
      .gw-fb-close:hover { background: #f3f4f6; color: #333; }

      .gw-fb-body { padding: 12px 16px; flex: 1; overflow-y: auto; }

      .gw-fb-chips { display: flex; gap: 6px; margin-bottom: 10px; }
      .gw-fb-chip {
        padding: 5px 12px; border-radius: 14px; font-size: 12px; font-weight: 600;
        cursor: pointer; border: 1.5px solid #ddd; background: none; color: #888;
        transition: all 0.15s;
      }
      .gw-fb-chip.active { border-color: #004B87; background: rgba(0,75,135,0.08); color: #004B87; }

      .gw-fb-textarea {
        width: 100%; min-height: 80px; max-height: 200px; padding: 10px 12px;
        border: 1.5px solid #e5e7eb; border-radius: 10px; font-size: 14px;
        font-family: inherit; resize: none; outline: none; box-sizing: border-box;
        line-height: 1.5;
      }
      .gw-fb-textarea:focus { border-color: #004B87; }
      .gw-fb-textarea::placeholder { color: #9ca3af; }

      .gw-fb-footer-bar {
        display: flex; align-items: center; justify-content: space-between;
        padding: 10px 16px; border-top: 1px solid #e5e7eb; flex-shrink: 0;
      }
      .gw-fb-hint { font-size: 11px; color: #9ca3af; }
      .gw-fb-send {
        background: #004B87; color: #fff; border: none; border-radius: 8px;
        padding: 8px 20px; font-size: 13px; font-weight: 600; cursor: pointer;
        transition: background 0.12s;
      }
      .gw-fb-send:hover { background: #003a6b; }
      .gw-fb-send:disabled { background: #7facc8; cursor: not-allowed; }

      .gw-fb-urgency {
        display: flex; align-items: center; gap: 8px; margin-bottom: 10px;
      }
      .gw-fb-urgency-label { font-size: 12px; color: #9ca3af; cursor: pointer; user-select: none; }
      .gw-fb-urgency-label.active { color: #004B87; font-weight: 600; }
      .gw-fb-toggle-track {
        width: 32px; height: 18px; border-radius: 9px; cursor: pointer;
        position: relative; transition: background-color 0.2s; flex-shrink: 0;
      }
      .gw-fb-toggle-thumb {
        width: 14px; height: 14px; border-radius: 50%; background: #fff;
        position: absolute; top: 2px; transition: left 0.2s;
        box-shadow: 0 1px 3px rgba(0,0,0,0.2);
      }

      .gw-fb-error { color: #d32f2f; font-size: 12px; margin-top: 4px; display: none; }
      .gw-fb-error.show { display: block; }

      .gw-fb-success { text-align: center; padding: 32px 16px; }
      .gw-fb-success-icon {
        width: 48px; height: 48px; margin: 0 auto 10px;
        background: #d1fae5; border-radius: 50%; display: flex;
        align-items: center; justify-content: center;
      }
      .gw-fb-success-icon svg { width: 24px; height: 24px; fill: #059669; }
      .gw-fb-success-msg { font-size: 15px; color: #374151; }
      .gw-fb-success-id { font-size: 12px; color: #9ca3af; margin-top: 4px; }

      .gw-fb-history-item {
        border: 1px solid #e5e7eb; border-radius: 10px; padding: 12px;
        margin-bottom: 10px;
      }
      .gw-fb-history-body { font-size: 13px; color: #374151; white-space: pre-wrap; word-break: break-word; line-height: 1.4; }
      .gw-fb-history-meta {
        font-size: 11px; color: #9ca3af; margin-top: 6px;
        display: flex; gap: 8px; flex-wrap: wrap; align-items: center;
      }
      .gw-fb-history-status {
        display: inline-block; padding: 2px 8px; border-radius: 8px;
        font-size: 11px; font-weight: 600;
      }
      .gw-fb-status-new { background: #dbeafe; color: #004B87; }
      .gw-fb-status-reviewed { background: #fef3c7; color: #92400e; }
      .gw-fb-status-roadmapped { background: #d1fae5; color: #065f46; }
      .gw-fb-status-done { background: #e5e7eb; color: #374151; }
      .gw-fb-history-outcome {
        font-size: 12px; color: #374151; margin-top: 6px; padding: 6px 8px;
        background: #f9fafb; border-radius: 6px; line-height: 1.4;
      }
      .gw-fb-history-outcome strong { color: #1a1a1a; }
      .gw-fb-empty { text-align: center; color: #9ca3af; padding: 24px 16px; font-size: 13px; }
      .gw-fb-link-btn {
        background: none; border: none; color: #004B87; font-size: 12px;
        cursor: pointer; text-decoration: underline; padding: 2px;
      }
    `;
    document.head.appendChild(css);
  }

  function init() {
    injectStyles();
    var container = document.createElement('div');
    container.id = WIDGET_ID;

    var fab = document.createElement('button');
    fab.id = 'gw-fb-fab';
    fab.title = 'Send Feedback';
    fab.innerHTML = '<svg viewBox="0 0 24 24"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H5.17L4 17.17V4h16v12zM7 9h2v2H7zm4 0h2v2h-2zm4 0h2v2h-2z"/></svg>';
    fab.onclick = function () { openModal(); };

    var backdrop = document.createElement('div');
    backdrop.id = 'gw-fb-backdrop';
    backdrop.onclick = closeModal;

    var modal = document.createElement('div');
    modal.id = 'gw-fb-modal';

    container.appendChild(fab);
    container.appendChild(backdrop);
    container.appendChild(modal);
    document.body.appendChild(container);

    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && isOpen) closeModal();
    });
  }

  function openModal() {
    isOpen = true;
    document.getElementById('gw-fb-fab').style.display = 'none';
    document.getElementById('gw-fb-backdrop').classList.add('open');
    var modal = document.getElementById('gw-fb-modal');
    modal.classList.add('open');
    modal.innerHTML = '';
    buildFormView(modal);
    var ta = modal.querySelector('.gw-fb-textarea');
    if (ta) setTimeout(function () { ta.focus(); }, 50);
  }

  function closeModal() {
    isOpen = false;
    document.getElementById('gw-fb-fab').style.display = 'flex';
    document.getElementById('gw-fb-backdrop').classList.remove('open');
    var modal = document.getElementById('gw-fb-modal');
    modal.classList.remove('open');
    modal.innerHTML = '';
  }

  function buildFormView(modal) {
    var selectedType = 'feature';
    var selectedUrgency = 'normal';

    // Header
    var header = document.createElement('div');
    header.className = 'gw-fb-header';

    var title = document.createElement('h2');
    title.textContent = 'Send Feedback';

    var historyBtn = document.createElement('button');
    historyBtn.className = 'gw-fb-close';
    historyBtn.title = 'Past submissions';
    historyBtn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';
    historyBtn.onclick = function () { showHistory(modal); };

    var closeBtn = document.createElement('button');
    closeBtn.className = 'gw-fb-close';
    closeBtn.innerHTML = '&times;';
    closeBtn.onclick = closeModal;

    header.appendChild(title);
    header.appendChild(historyBtn);
    header.appendChild(closeBtn);

    // Body
    var body = document.createElement('div');
    body.className = 'gw-fb-body';

    // Type chips
    var chips = document.createElement('div');
    chips.className = 'gw-fb-chips';
    var types = [
      { key: 'feature', label: 'Idea' },
      { key: 'bug', label: 'Bug' },
      { key: 'other', label: 'Other' },
    ];
    var chipBtns = [];
    types.forEach(function (t) {
      var chip = document.createElement('button');
      chip.className = 'gw-fb-chip' + (t.key === selectedType ? ' active' : '');
      chip.textContent = t.label;
      chip.onclick = function () {
        selectedType = t.key;
        chipBtns.forEach(function (c, i) {
          c.className = 'gw-fb-chip' + (types[i].key === selectedType ? ' active' : '');
        });
        updatePlaceholder();
      };
      chipBtns.push(chip);
      chips.appendChild(chip);
    });

    // Urgency toggle
    var urgRow = document.createElement('div');
    urgRow.className = 'gw-fb-urgency';

    var urgLabel1 = document.createElement('span');
    urgLabel1.className = 'gw-fb-urgency-label';
    urgLabel1.textContent = 'Normal';

    var track = document.createElement('div');
    track.className = 'gw-fb-toggle-track';
    var thumb = document.createElement('div');
    thumb.className = 'gw-fb-toggle-thumb';
    track.appendChild(thumb);

    var urgLabel2 = document.createElement('span');
    urgLabel2.className = 'gw-fb-urgency-label';
    urgLabel2.textContent = 'Urgent';

    function updateUrgencyUI() {
      var isUrgent = selectedUrgency === 'critical';
      track.style.backgroundColor = isUrgent ? '#004B87' : '#e5e7eb';
      thumb.style.left = isUrgent ? '16px' : '2px';
      urgLabel1.className = 'gw-fb-urgency-label' + (!isUrgent ? ' active' : '');
      urgLabel2.className = 'gw-fb-urgency-label' + (isUrgent ? ' active' : '');
    }
    updateUrgencyUI();

    function toggleUrgency() {
      selectedUrgency = selectedUrgency === 'critical' ? 'normal' : 'critical';
      updateUrgencyUI();
    }
    track.onclick = toggleUrgency;
    urgLabel1.onclick = function () { selectedUrgency = 'normal'; updateUrgencyUI(); };
    urgLabel2.onclick = function () { selectedUrgency = 'critical'; updateUrgencyUI(); };

    urgRow.appendChild(urgLabel1);
    urgRow.appendChild(track);
    urgRow.appendChild(urgLabel2);

    // Textarea
    var textarea = document.createElement('textarea');
    textarea.className = 'gw-fb-textarea';
    textarea.setAttribute('maxlength', '10000');

    function updatePlaceholder() {
      var placeholders = {
        bug: 'What went wrong?',
        feature: 'What would make this better?',
        other: "What's on your mind?"
      };
      textarea.placeholder = placeholders[selectedType] || placeholders.other;
    }
    updatePlaceholder();

    var errorMsg = document.createElement('div');
    errorMsg.className = 'gw-fb-error';

    textarea.addEventListener('input', function () {
      if (this.value.length > 0) errorMsg.classList.remove('show');
    });

    // Enter to submit, Shift+Enter for newline
    textarea.addEventListener('keydown', function (e) {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        doSubmit();
      }
    });

    body.appendChild(chips);
    body.appendChild(urgRow);
    body.appendChild(textarea);
    body.appendChild(errorMsg);

    // Footer with hint + send button
    var footer = document.createElement('div');
    footer.className = 'gw-fb-footer-bar';

    var hint = document.createElement('span');
    hint.className = 'gw-fb-hint';
    hint.textContent = 'Enter to send \u00B7 Shift+Enter for new line';

    var sendBtn = document.createElement('button');
    sendBtn.className = 'gw-fb-send';
    sendBtn.textContent = 'Send';
    sendBtn.onclick = function () { doSubmit(); };

    footer.appendChild(hint);
    footer.appendChild(sendBtn);

    function doSubmit() {
      var text = textarea.value.trim();
      if (!text) {
        errorMsg.textContent = "Tell us what's on your mind.";
        errorMsg.classList.add('show');
        textarea.focus();
        return;
      }
      sendBtn.disabled = true;
      sendBtn.textContent = '...';

      fetch('/api/feedback', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body: text, type: selectedType, urgency: selectedUrgency }),
      })
        .then(function (r) { return r.json(); })
        .then(function (data) {
          if (data.success) {
            // Keep modal open, reset form, show inline confirmation
            textarea.value = '';
            sendBtn.disabled = false;
            sendBtn.textContent = 'Send';
            hint.textContent = '\u2713 Sent! Enter another or close.';
            hint.style.color = '#059669';
            setTimeout(function () {
              hint.textContent = 'Enter to send \u00B7 Shift+Enter for new line';
              hint.style.color = '';
            }, 3000);
            textarea.focus();
          } else {
            sendBtn.disabled = false;
            sendBtn.textContent = 'Send';
            errorMsg.textContent = data.error || 'Something went wrong';
            errorMsg.classList.add('show');
          }
        })
        .catch(function () {
          sendBtn.disabled = false;
          sendBtn.textContent = 'Send';
          errorMsg.textContent = 'Network error — please try again.';
          errorMsg.classList.add('show');
        });
    }

    modal.appendChild(header);
    modal.appendChild(body);
    modal.appendChild(footer);
  }

  function showSuccess(modal, feedbackId, message) {
    modal.innerHTML = '';

    var header = document.createElement('div');
    header.className = 'gw-fb-header';
    var t = document.createElement('h2');
    t.textContent = 'Send Feedback';
    header.appendChild(t);
    var closeBtn = document.createElement('button');
    closeBtn.className = 'gw-fb-close';
    closeBtn.textContent = '\u00D7';
    closeBtn.onclick = closeModal;
    header.appendChild(closeBtn);

    var success = document.createElement('div');
    success.className = 'gw-fb-success';

    var icon = document.createElement('div');
    icon.className = 'gw-fb-success-icon';
    icon.innerHTML = '<svg viewBox="0 0 24 24"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z"/></svg>';

    var msg = document.createElement('div');
    msg.className = 'gw-fb-success-msg';
    msg.textContent = message || 'Thank you for your feedback!';

    var idLine = document.createElement('div');
    idLine.className = 'gw-fb-success-id';
    idLine.textContent = feedbackId ? 'Reference: #' + feedbackId : '';

    success.appendChild(icon);
    success.appendChild(msg);
    success.appendChild(idLine);

    modal.appendChild(header);
    modal.appendChild(success);

    setTimeout(function () { closeModal(); }, 2500);
  }

  function showHistory(modal) {
    modal.innerHTML = '';

    var header = document.createElement('div');
    header.className = 'gw-fb-header';
    var t = document.createElement('h2');
    t.textContent = 'My Submissions';
    header.appendChild(t);
    var closeBtn = document.createElement('button');
    closeBtn.className = 'gw-fb-close';
    closeBtn.textContent = '\u00D7';
    closeBtn.onclick = closeModal;
    header.appendChild(closeBtn);

    var body = document.createElement('div');
    body.className = 'gw-fb-body';
    body.innerHTML = '<div class="gw-fb-empty">Loading...</div>';

    var footer = document.createElement('div');
    footer.className = 'gw-fb-footer-bar';
    var backBtn = document.createElement('button');
    backBtn.className = 'gw-fb-link-btn';
    backBtn.textContent = '\u2190 New feedback';
    backBtn.onclick = function () {
      modal.innerHTML = '';
      buildFormView(modal);
      var ta = modal.querySelector('.gw-fb-textarea');
      if (ta) setTimeout(function () { ta.focus(); }, 50);
    };
    footer.appendChild(backBtn);

    modal.appendChild(header);
    modal.appendChild(body);
    modal.appendChild(footer);

    fetch('/api/feedback', { credentials: 'same-origin' })
      .then(function (r) { return r.json(); })
      .then(function (items) {
        body.innerHTML = '';
        if (!items || items.length === 0) {
          body.innerHTML = '<div class="gw-fb-empty">No submissions yet.</div>';
          return;
        }
        items.forEach(function (item) {
          var el = document.createElement('div');
          el.className = 'gw-fb-history-item';

          var bodyText = document.createElement('div');
          bodyText.className = 'gw-fb-history-body';
          bodyText.textContent = item.body;

          var meta = document.createElement('div');
          meta.className = 'gw-fb-history-meta';

          var statusSpan = document.createElement('span');
          statusSpan.className = 'gw-fb-history-status gw-fb-status-' + item.status;
          statusSpan.textContent = statusLabels[item.status] || item.status;
          meta.appendChild(statusSpan);

          var typeSpan = document.createElement('span');
          typeSpan.textContent = item.type;
          typeSpan.style.textTransform = 'capitalize';
          meta.appendChild(typeSpan);

          var dateSpan = document.createElement('span');
          try {
            var d = new Date(item.created_at.includes('Z') ? item.created_at : item.created_at + 'Z');
            dateSpan.textContent = d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
          } catch (e) { dateSpan.textContent = item.created_at; }
          meta.appendChild(dateSpan);

          el.appendChild(bodyText);
          el.appendChild(meta);

          if (item.admin_notes) {
            var outcome = document.createElement('div');
            outcome.className = 'gw-fb-history-outcome';
            var lbl = document.createElement('strong');
            lbl.textContent = 'Response: ';
            outcome.appendChild(lbl);
            outcome.appendChild(document.createTextNode(item.admin_notes));
            el.appendChild(outcome);
          }

          body.appendChild(el);
        });
      })
      .catch(function () {
        body.innerHTML = '<div class="gw-fb-empty">Failed to load. Try again.</div>';
      });
  }

  // init() is called by checkAuth() when user has feedback access
})();
