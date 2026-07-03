(function() {
    'use strict';

    // --- STATE ---
    const state = {
        token: localStorage.getItem('fc_token') || null,
        user: JSON.parse(localStorage.getItem('fc_user') || 'null'),
        room: { id: 1, name: 'General' },
        ws: null,
        pendingMedia: null // Holds uploaded media data before sending via WS
    };

    // --- DOM ELEMENTS ---
    const dom = {
        viewAuth: document.getElementById('view-auth'),
        viewChat: document.getElementById('view-chat'),
        formGuest: document.getElementById('form-guest'),
        formLogin: document.getElementById('form-login'),
        formChat: document.getElementById('form-chat'),
        inputGuestName: document.getElementById('input-guest-name'),
        inputGuestGender: document.getElementById('input-guest-gender'),
        inputGuestAge: document.getElementById('input-guest-age'),
        inputGuestAbout: document.getElementById('input-guest-about'),
        inputEmail: document.getElementById('input-email'),
        inputPassword: document.getElementById('input-password'),
        inputMessage: document.getElementById('input-message'),
        inputMedia: document.getElementById('input-media'),
        btnToggleGuestOptions: document.getElementById('btn-toggle-guest-options'),
        guestOptions: document.getElementById('guest-options'),
        btnToggleAuth: document.getElementById('btn-toggle-auth'),
        btnAttach: document.getElementById('btn-attach'),
        btnCancelMedia: document.getElementById('btn-cancel-media'),
        mediaPreviewContainer: document.getElementById('media-preview-container'),
        mediaPreviewContent: document.getElementById('media-preview-content'),
        btnToggleSidebar: document.getElementById('btn-toggle-sidebar'),
        btnCloseSidebar: document.getElementById('btn-close-sidebar'),
        btnLogout: document.getElementById('btn-logout'),
        btnLogoutDesktop: document.getElementById('btn-logout-desktop'),
        sidebarOverlay: document.getElementById('sidebar-overlay'),
        chatMessages: document.getElementById('chat-messages'),
        systemAlerts: document.getElementById('system-alerts'),
        mobileRoomName: document.getElementById('mobile-room-name'),
        roomNameDisplay: document.getElementById('room-name-display'),
        userCountDisplay: document.getElementById('user-count-display'),
        userNameDisplay: document.getElementById('user-name-display'),
        userTypeDisplay: document.getElementById('user-type-display'),
        toastContainer: document.getElementById('toast-container')
    };

    // --- INITIALIZATION ---
    function init() {
        if (state.token && state.user) {
            showChatView();
            connectWebSocket();
        } else {
            showAuthView();
        }
        bindEvents();
    }

    function bindEvents() {
        dom.formGuest.addEventListener('submit', handleGuestSubmit);
        dom.formLogin.addEventListener('submit', handleLoginSubmit);
        dom.formChat.addEventListener('submit', handleChatSubmit);
        
        dom.btnToggleGuestOptions.addEventListener('click', () => {
            dom.guestOptions.classList.toggle('open', !dom.guestOptions.classList.contains('open'));
            dom.btnToggleGuestOptions.textContent = dom.guestOptions.classList.contains('open') ? '- Less Options' : '+ More Options';
        });

        dom.btnToggleAuth.addEventListener('click', toggleAuthForm);
        dom.btnAttach.addEventListener('click', () => dom.inputMedia.click());
        dom.inputMedia.addEventListener('change', handleMediaUpload);
        dom.btnCancelMedia.addEventListener('click', clearMediaPreview);
        
        dom.btnToggleSidebar.addEventListener('click', () => dom.sidebar.classList.add('open'));
        dom.btnCloseSidebar.addEventListener('click', closeSidebar);
        dom.sidebarOverlay.addEventListener('click', closeSidebar);
        
        dom.btnLogout.addEventListener('click', handleLogout);
        dom.btnLogoutDesktop.addEventListener('click', handleLogout);
    }

    // --- SPEC COMPLIANT FETCH HELPERS ---
    // Spec Note: Go's http.Error returns text/plain, not JSON. We must handle both.
    async function apiRequest(url, options = {}) {
        try {
            const res = await fetch(url, options);
            
            if (!res.ok) {
                // Try to parse as JSON, fallback to text
                let errorMsg = `Error ${res.status}`;
                try {
                    const errData = await res.json();
                    errorMsg = errData.error || JSON.stringify(errData);
                } catch (e) {
                    errorMsg = await res.text() || errorMsg;
                }
                throw new Error(errorMsg);
            }

            // Handle 204 No Content
            if (res.status === 204) return null;

            return await res.json();
        } catch (err) {
            showToast(err.message, 'error');
            throw err;
        }
    }

    // --- VIEW MANAGEMENT ---
    function showAuthView() {
        dom.viewAuth.classList.add('active');
        dom.viewChat.classList.remove('active');
    }

    function showChatView() {
        dom.viewAuth.classList.remove('active');
        dom.viewChat.classList.add('active');
        dom.userNameDisplay.textContent = state.user.username;
        dom.userTypeDisplay.textContent = state.user.UserType || 'guest';
        dom.mobileRoomName.textContent = state.room.name;
        dom.roomNameDisplay.textContent = state.room.name;
        dom.inputMessage.focus();
    }

    function closeSidebar() { dom.sidebar.classList.remove('open'); }

    function toggleAuthForm() {
        const isLoginVisible = !dom.formLogin.classList.contains('hidden');
        dom.formLogin.classList.toggle('hidden', isLoginVisible);
        dom.formGuest.classList.toggle('hidden', !isLoginVisible);
        dom.btnToggleAuth.textContent = isLoginVisible ? 'Create an Account' : 'Have an account? Login';
    }

    // --- AUTH HANDLERS (Strict Spec Implementation) ---

    // Spec: POST /auth/guest -> 201 { token, user }
    async function handleGuestSubmit(e) {
        e.preventDefault();
        const payload = {
            username: dom.inputGuestName.value.trim(),
            gender: dom.inputGuestGender.value,
            age: parseInt(dom.inputGuestAge.value) || 0,
            about: dom.inputGuestAbout.value.trim()
        };

        const data = await apiRequest('/auth/guest', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        setAuth(data.token, data.user);
        showChatView();
        connectWebSocket();
    }

    // Spec: POST /auth/register -> 201 (User Object Only, NO TOKEN)
    async function handleLoginSubmit(e) {
        e.preventDefault();
        const payload = {
            email: dom.inputEmail.value.trim(),
            password: dom.inputPassword.value
        };

        const data = await apiRequest('/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        setAuth(data.token, data.user);
        showChatView();
        connectWebSocket();
    }

    // Note: If you implement the Register button later, remember the spec dictates:
    // POST /auth/register returns 201 User object, but NO token.
    // You must show a success toast and switch to the login form.

    function handleLogout() {
        clearAuth();
        if (state.ws) state.ws.close();
        showAuthView();
    }

    function setAuth(token, user) {
        state.token = token;
        state.user = user;
        localStorage.setItem('fc_token', token);
        localStorage.setItem('fc_user', JSON.stringify(user));
    }

    function clearAuth() {
        state.token = null;
        state.user = null;
        localStorage.removeItem('fc_token');
        localStorage.removeItem('fc_user');
        dom.chatMessages.innerHTML = '';
    }

    // --- WEBSOCKET (Strict Spec Implementation) ---

    function connectWebSocket() {
        if (state.ws) state.ws.close();

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws?token=${state.token}`;
        
        state.ws = new WebSocket(wsUrl);

        state.ws.onopen = () => {
            // Spec: Send join message to trigger history fetch
            sendWsMessage({ type: 'join', room_id: state.room.id });
            dom.userCountDisplay.textContent = 'Connected';
        };

        state.ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                handleWsMessage(msg);
            } catch (e) {
                console.error('Invalid WS message format', e);
            }
        };

        state.ws.onclose = () => {
            dom.userCountDisplay.textContent = 'Disconnected';
        };
    }

    function sendWsMessage(payload) {
        if (state.ws && state.ws.readyState === WebSocket.OPEN) {
            state.ws.send(JSON.stringify(payload));
        }
    }

    function handleWsMessage(msg) {
        switch (msg.type) {
            // Spec: Direct guest creation response
            case 'auth':
                if (msg.token) setAuth(msg.token, state.user);
                break;

            // Spec: Rate limit info on connect
            case 'rate_limit':
                // Could update UI to show remaining messages if low
                break;

            // Spec: History payload after joining room
            case 'history':
                renderHistory(msg.messages);
                break;

            // Spec: Incoming text or media chat message
            case 'chat':
            case 'media':
                appendMessage(msg, msg.sender_id === state.user.ID);
                break;

            // Spec: Error payloads
            case 'error':
                showAlert(msg.error, 'error');
                if (msg.code === 'RATE_LIMITED') {
                    dom.inputMessage.disabled = true;
                    setTimeout(() => dom.inputMessage.disabled = false, 3000); // Cooldown
                }
                break;
        }
    }

    // --- CHAT LOGIC ---

    function handleChatSubmit(e) {
        e.preventDefault();
        const content = dom.inputMessage.value.trim();
        if (!content && !state.pendingMedia) return;

        // Spec: Send standard chat or media payload
        const payload = { type: 'chat', room_id: state.room.id };
        
        if (state.pendingMedia) {
            payload.type = 'media';
            payload.media_id = state.pendingMedia.id;
            clearMediaPreview();
        } else {
            payload.content = content;
        }

        sendWsMessage(payload);
        dom.inputMessage.value = '';
        dom.inputMessage.focus();
    }

    // Spec: Media Upload Endpoints
    async function handleMediaUpload(e) {
        const file = e.target.files[0];
        if (!file) return;

        let endpoint = '/media/upload/image';
        let fieldName = 'image';
        
        if (file.type === 'image/gif') {
            endpoint = '/media/upload/gif';
            fieldName = 'gif';
        } else if (file.type.startsWith('audio/')) {
            endpoint = '/media/upload/voice';
            fieldName = 'voice';
        } else if (!file.type.startsWith('image/')) {
            showToast('Invalid file type.', 'error');
            return;
        }

        const formData = new FormData();
        formData.append(fieldName, file);

        // Show local preview immediately (UX feedback)
        showMediaPreview(file);

        try {
            // Spec: Auth via Bearer token
            const data = await apiRequest(endpoint, {
                method: 'POST',
                headers: { 'Authorization': `Bearer ${state.token}` },
                body: formData
            });

            // Spec: Save the returned ID to attach to the WS payload
            state.pendingMedia = { id: data.id, url: data.url, type: data.type };
            
        } catch (err) {
            clearMediaPreview(); // Remove preview on failure
        } finally {
            dom.inputMedia.value = '';
        }
    }

    // --- UI RENDERING ---

    function showMediaPreview(file) {
        dom.mediaPreviewContainer.classList.remove('hidden');
        dom.mediaPreviewContent.innerHTML = '';
        
        if (file.type.startsWith('image/')) {
            const img = document.createElement('img');
            img.src = URL.createObjectURL(file);
            dom.mediaPreviewContent.appendChild(img);
        } else if (file.type.startsWith('audio/')) {
            const audio = document.createElement('audio');
            audio.src = URL.createObjectURL(file);
            audio.controls = true;
            dom.mediaPreviewContent.appendChild(audio);
        }
    }

    function clearMediaPreview() {
        state.pendingMedia = null;
        dom.mediaPreviewContainer.classList.add('hidden');
        dom.mediaPreviewContent.innerHTML = '';
    }

    function renderHistory(messages) {
        dom.chatMessages.innerHTML = '';
        // Spec: Messages come in DESC order from server, reverse for UI
        messages.reverse().forEach(msg => {
            appendMessage(msg, msg.sender_id === state.user.ID);
        });
    }

    function appendMessage(msg, isSent) {
        const wrapper = document.createElement('div');
        wrapper.className = `message ${isSent ? 'sent' : 'received'}`;

        let mediaHtml = '';
        // Spec: Handle media_url and media_type from WS payload
        if (msg.media_url) {
            if (msg.media_type === 'voice') {
                mediaHtml = `<audio controls src="${msg.media_url}" class="message-media"></audio>`;
            } else {
                mediaHtml = `<img src="${msg.media_url}" class="message-media" alt="Shared media" loading="lazy">`;
            }
        }

        // Spec: content might be mutated if filtered=true
        const contentHtml = msg.content ? `<p>${escapeHtml(msg.content)}</p>` : '';
        const time = msg.timestamp ? new Date(msg.timestamp * 1000).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}) : '';

        wrapper.innerHTML = `
            ${mediaHtml}
            <div class="message-bubble">
                ${!isSent ? `<strong style="display:block; margin-bottom:4px; font-size:0.85rem; color:var(--color-primary);">${escapeHtml(msg.username)}</strong>` : ''}
                ${contentHtml}
            </div>
            <div class="message-meta">
                ${time}
                ${msg.filtered ? '<span style="color:var(--color-error); font-weight:600;">Filtered</span>' : ''}
            </div>
        `;

        dom.chatMessages.appendChild(wrapper);
        scrollToBottom();
    }

    function scrollToBottom() {
        setTimeout(() => { dom.chatMessages.scrollTop = dom.chatMessages.scrollHeight; }, 50);
    }

    function showAlert(message, type = 'error') {
        const alert = document.createElement('div');
        alert.className = `alert ${type}`;
        alert.textContent = message;
        dom.systemAlerts.appendChild(alert);
        dom.systemAlerts.classList.add('active');
        setTimeout(() => {
            alert.remove();
            if (dom.systemAlerts.children.length === 0) dom.systemAlerts.classList.remove('active');
        }, 4000);
    }

    function showToast(message, type = 'error') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        dom.toastContainer.appendChild(toast);
        setTimeout(() => toast.remove(), 4000);
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    init();
})();