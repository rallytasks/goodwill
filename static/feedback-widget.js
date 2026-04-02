// Goodwill Feedback Widget (FAB)
// Floating action button for feature requests, bugs, and feedback.
// Adapted from PennyAI's feedback widget.
(function () {
  'use strict';

  var WIDGET_ID = 'goodwill-feedback-widget';
  if (document.getElementById(WIDGET_ID)) return;

  var isOpen = false;
  var canSubmit = false;

  // Check if current user has feedback access
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
    css.textContent = [
      '#gw-fb-fab {',
      '  position: fixed; bottom: 24px; right: 24px; z-index: 99999;',
      '  width: 48px; height: 48px; border-radius: 50%;',
      '  background: #004B87; color: #fff; border: none; cursor: pointer;',
      '  box-shadow: 0 4px 14px rgba(0,75,135,0.4);',
      '  display: flex; align-items: center; justify-content: center;',
      '  transition: transform 0.15s ease, box-shadow 0.15s ease;',
      '}',
      '#gw-fb-fab:hover { transform: scale(1.08); box-shadow: 0 6px 20px rgba(0,75,135,0.5); }',
      '#gw-fb-fab:active { transform: scale(0.95); }',
      '#gw-fb-fab svg { width: 22px; height: 22px; fill: currentColor; }',
      '',
      '#gw-fb-overlay {',
      '  position: fixed; inset: 0; z-index: 100000;',
      '  background: #fff; display: none;',
      '  flex-direction: column;',
      '}',
      '#gw-fb-overlay.open { display: flex; }',
      '',
      '.gw-fb-header {',
      '  display: flex; align-items: center; justify-content: space-between;',
      '  padding: 12px 16px; border-bottom: 1px solid #e5e7eb; flex-shrink: 0;',
      '  gap: 8px;',
      '}',
      '.gw-fb-header h2 { margin: 0; font-size: 16px; font-weight: 600; color: #1a1a1a; flex: 1; }',
      '.gw-fb-close {',
      '  background: none; border: none; cursor: pointer; font-size: 22px;',
      '  color: #999; padding: 4px 8px; border-radius: 6px; line-height: 1; flex-shrink: 0;',
      '}',
      '.gw-fb-close:hover { background: #f3f4f6; color: #333; }',
      '.gw-fb-send {',
      '  background: #004B87; color: #fff; border: none; border-radius: 8px;',
      '  padding: 6px 16px; font-size: 14px; font-weight: 600; cursor: pointer;',
      '  flex-shrink: 0; transition: background 0.12s;',
      '}',
      '.gw-fb-send:hover { background: #003a6b; }',
      '.gw-fb-send:disabled { background: #7facc8; cursor: not-allowed; }',
      '',
      '.gw-fb-priority-row {',
      '  display: flex; align-items: center; gap: 10px; padding: 10px 16px; flex-shrink: 0;',
      '  border-bottom: 1px solid #f3f4f6;',
      '}',
      '.gw-fb-toggle-track {',
      '  width: 36px; height: 20px; border-radius: 10px; cursor: pointer;',
      '  position: relative; transition: background-color 0.2s; flex-shrink: 0;',
      '}',
      '.gw-fb-toggle-thumb {',
      '  width: 16px; height: 16px; border-radius: 50%; background: #fff;',
      '  position: absolute; top: 2px; transition: left 0.2s;',
      '  box-shadow: 0 1px 3px rgba(0,0,0,0.2);',
      '}',
      '.gw-fb-toggle-label {',
      '  font-size: 13px; font-weight: 600; cursor: pointer; user-select: none;',
      '}',
      '',
      '.gw-fb-body {',
      '  padding: 12px 16px; flex: 1; display: flex; flex-direction: column;',
      '  max-width: 600px; width: 100%; margin: 0 auto; box-sizing: border-box;',
      '}',
      '',
      '.gw-fb-chips { display: flex; gap: 6px; margin-bottom: 12px; }',
      '.gw-fb-chip {',
      '  padding: 6px 14px; border-radius: 16px; font-size: 13px; font-weight: 600;',
      '  cursor: pointer; border: 1.5px solid #ddd; background: none; color: #888;',
      '  transition: all 0.15s;',
      '}',
      '.gw-fb-chip.active { border-color: #004B87; background: rgba(0,75,135,0.08); color: #004B87; }',
      '',
      '.gw-fb-textarea {',
      '  width: 100%; flex: 1; min-height: 80px; padding: 12px;',
      '  border: 1.5px solid #e5e7eb; border-radius: 12px; font-size: 16px;',
      '  font-family: inherit; resize: none; outline: none;',
      '  transition: border-color 0.15s; box-sizing: border-box;',
      '}',
      '.gw-fb-textarea:focus { border-color: #004B87; }',
      '.gw-fb-textarea::placeholder { color: #9ca3af; }',
      '',
      '.gw-fb-error { color: #d32f2f; font-size: 13px; margin-top: 6px; display: none; }',
      '.gw-fb-error.show { display: block; }',
      '',
      '.gw-fb-success { text-align: center; padding: 60px 24px; flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; }',
      '.gw-fb-success-icon {',
      '  width: 64px; height: 64px; margin: 0 auto 16px;',
      '  background: #d1fae5; border-radius: 50%; display: flex;',
      '  align-items: center; justify-content: center;',
      '}',
      '.gw-fb-success-icon svg { width: 32px; height: 32px; fill: #059669; }',
      '.gw-fb-success-msg { font-size: 17px; color: #374151; line-height: 1.5; }',
      '.gw-fb-success-id { font-size: 13px; color: #9ca3af; margin-top: 6px; }',
      '',
      '.gw-fb-footer {',
      '  padding: 10px 16px; text-align: center; flex-shrink: 0;',
      '  border-top: 1px solid #e5e7eb;',
      '}',
      '.gw-fb-toggle-history {',
      '  background: none; border: none; color: #6b7280; font-size: 13px;',
      '  cursor: pointer; text-decoration: underline; padding: 4px;',
      '}',
      '.gw-fb-toggle-history:hover { color: #004B87; }',
      '',
      '.gw-fb-history-item {',
      '  border: 1px solid #e5e7eb; border-radius: 12px; padding: 14px;',
      '  margin-bottom: 12px;',
      '}',
      '.gw-fb-history-body { font-size: 15px; color: #374151; white-space: pre-wrap; word-break: break-word; line-height: 1.5; }',
      '.gw-fb-history-meta {',
      '  font-size: 12px; color: #9ca3af; margin-top: 8px;',
      '  display: flex; gap: 10px; flex-wrap: wrap; align-items: center;',
      '}',
      '.gw-fb-history-status {',
      '  display: inline-block; padding: 3px 10px; border-radius: 10px;',
      '  font-size: 12px; font-weight: 600;',
      '}',
      '.gw-fb-status-new { background: #dbeafe; color: #004B87; }',
      '.gw-fb-status-reviewed { background: #fef3c7; color: #92400e; }',
      '.gw-fb-status-roadmapped { background: #d1fae5; color: #065f46; }',
      '.gw-fb-status-done { background: #e5e7eb; color: #374151; }',
      '.gw-fb-history-outcome {',
      '  font-size: 13px; color: #374151; margin-top: 8px; padding: 8px 10px;',
      '  background: #f9fafb; border-radius: 8px; line-height: 1.4;',
      '}',
      '.gw-fb-history-outcome strong { color: #1a1a1a; }',
      '.gw-fb-empty { text-align: center; color: #9ca3af; padding: 40px 24px; font-size: 15px; }',
    ].join('\n');
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

    var overlay = document.createElement('div');
    overlay.id = 'gw-fb-overlay';

    container.appendChild(fab);
    container.appendChild(overlay);
    document.body.appendChild(container);

    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && isOpen) closeModal();
    });
  }

  function openModal() {
    isOpen = true;
    document.getElementById('gw-fb-fab').style.display = 'none';
    var overlay = document.getElementById('gw-fb-overlay');
    overlay.classList.add('open');
    overlay.innerHTML = '';
    overlay.appendChild(buildFormView());
    var ta = overlay.querySelector('.gw-fb-textarea');
    if (ta) setTimeout(function () { ta.focus(); }, 50);
  }

  function closeModal() {
    isOpen = false;
    document.getElementById('gw-fb-fab').style.display = 'flex';
    var overlay = document.getElementById('gw-fb-overlay');
    overlay.classList.remove('open');
    overlay.innerHTML = '';
  }

  function buildFormView() {
    var modal = document.createElement('div');
    modal.style.cssText = 'display:flex;flex-direction:column;height:100%;';

    var selectedType = 'feature';
    var selectedUrgency = 'normal';

    // Header
    var header = document.createElement('div');
    header.className = 'gw-fb-header';

    var closeBtn = document.createElement('button');
    closeBtn.className = 'gw-fb-close';
    closeBtn.innerHTML = '&times;';
    closeBtn.onclick = closeModal;

    var title = document.createElement('h2');
    title.textContent = 'Send Feedback';

    var historyBtn = document.createElement('button');
    historyBtn.className = 'gw-fb-close';
    historyBtn.title = 'Past submissions';
    historyBtn.innerHTML = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';
    historyBtn.onclick = function () { showHistory(); };

    var sendBtn = document.createElement('button');
    sendBtn.className = 'gw-fb-send';
    sendBtn.textContent = 'Send';

    header.appendChild(closeBtn);
    header.appendChild(title);
    header.appendChild(historyBtn);
    header.appendChild(sendBtn);

    // Priority toggle
    var priorityRow = document.createElement('div');
    priorityRow.className = 'gw-fb-priority-row';

    var track = document.createElement('div');
    track.className = 'gw-fb-toggle-track';
    var thumb = document.createElement('div');
    thumb.className = 'gw-fb-toggle-thumb';
    track.appendChild(thumb);

    var label = document.createElement('span');
    label.className = 'gw-fb-toggle-label';

    function updateToggleUI() {
      var isUrgent = selectedUrgency === 'critical';
      track.style.backgroundColor = isUrgent ? '#004B87' : '#e5e7eb';
      thumb.style.left = isUrgent ? '18px' : '2px';
      label.textContent = isUrgent ? 'Urgent' : 'Normal';
      label.style.color = isUrgent ? '#004B87' : '#9ca3af';
    }
    updateToggleUI();

    function togglePriority() {
      selectedUrgency = selectedUrgency === 'critical' ? 'normal' : 'critical';
      updateToggleUI();
    }
    track.onclick = togglePriority;
    label.onclick = togglePriority;

    priorityRow.appendChild(track);
    priorityRow.appendChild(label);

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
    textarea.addEventListener('keydown', function (e) {
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        sendBtn.click();
      }
    });

    body.appendChild(chips);
    body.appendChild(textarea);
    body.appendChild(errorMsg);

    sendBtn.onclick = function () {
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
            showSuccess(data.id, data.message);
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
    };

    modal.appendChild(header);
    modal.appendChild(priorityRow);
    modal.appendChild(body);
    return modal;
  }

  function showSuccess(feedbackId, message) {
    var overlay = document.getElementById('gw-fb-overlay');
    overlay.innerHTML = '';

    var modal = document.createElement('div');
    modal.style.cssText = 'display:flex;flex-direction:column;height:100%;';

    var header = document.createElement('div');
    header.className = 'gw-fb-header';
    var t = document.createElement('h2');
    t.textContent = 'Send Feedback';
    t.style.flex = '1';
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
    overlay.appendChild(modal);

    setTimeout(function () { closeModal(); }, 3000);
  }

  function showHistory() {
    var overlay = document.getElementById('gw-fb-overlay');
    overlay.innerHTML = '';

    var modal = document.createElement('div');
    modal.style.cssText = 'display:flex;flex-direction:column;height:100%;';

    var header = document.createElement('div');
    header.className = 'gw-fb-header';
    var t = document.createElement('h2');
    t.style.flex = '1';
    t.textContent = 'My Submissions';
    header.appendChild(t);
    var closeBtn = document.createElement('button');
    closeBtn.className = 'gw-fb-close';
    closeBtn.textContent = '\u00D7';
    closeBtn.onclick = closeModal;
    header.appendChild(closeBtn);

    var body = document.createElement('div');
    body.className = 'gw-fb-body';
    body.style.overflowY = 'auto';
    body.innerHTML = '<div class="gw-fb-empty">Loading...</div>';

    var footer = document.createElement('div');
    footer.className = 'gw-fb-footer';
    var backBtn = document.createElement('button');
    backBtn.className = 'gw-fb-toggle-history';
    backBtn.textContent = 'New feedback';
    backBtn.onclick = function () { openModal(); };
    footer.appendChild(backBtn);

    modal.appendChild(header);
    modal.appendChild(body);
    modal.appendChild(footer);
    overlay.appendChild(modal);

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
          } catch (e) {
            dateSpan.textContent = item.created_at;
          }
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
