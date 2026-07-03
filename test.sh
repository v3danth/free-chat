mkdir -p web/static && cat << 'EOF' > web/static/index.html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Free Chat</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <!-- AUTH VIEW -->
    <div id="auth-view" class="view">
        <div class="auth-card">
            <h1 class="logo">free.chat</h1>
            <p class="tagline">Connect instantly. No strings attached.</p>
            
            <div class="auth-divider">
                <span>Quickest Entry</span>
            </div>

            <form id="guest-form" class="auth-form">
                <input type="text" id="guest-username" placeholder="Pick a temporary username" required maxlength="32" autocomplete="off">
                <button type="submit" class="btn btn-primary btn-full">Start Chatting Anonymously</button>
            </form>

            <div class="auth-divider">
                <span>Or use an account</span>
            </div>

            <form id="login-form" class="auth-form" style="display: none;">
                <input type="email" id="login-email" placeholder="Email" required>
                <input type="password" id="login-password" placeholder="Password" required>
                <button type="submit" class="btn btn-secondary btn-full">Login</button>
                <p class="switch-view">Don't have an account? <a href="#" id="show-register">Register</a></p>
            </form>

            <form id="register-form" class="auth-form" style="display: none;">
                <input type="text" id="reg-username" placeholder="Username" required maxlength="32">
                <input type="email" id="reg-email" placeholder="Email" required>
                <input type="password" id="reg-password" placeholder="Password" required>
                <button type="submit" class="btn btn-secondary btn-full">Create Account</button>
                <p class="switch-view">Already have an account? <a href="#" id="show-login">Login</a></p>
            </form>

            <a href="#" id="show-login-btn" class="link-btn">I have an account</a>
            <a href="#" id="show-guest-btn" class="link-btn" style="display: none;">Back to Guest Login</a>
        </div>
    </div>

    <!-- CHAT VIEW -->
    <div id="chat-view" class="view" style="display: none;">
        <header class="chat-header">
            <div class="room-info">
                <h2 id="room-name">General Room</h2>
                <span id="user-status" class="status-badge">Connected</span>
            </div>
            <div class="user-menu">
                <span id="display-username" class="username-tag">User</span>
                <button id="logout-btn" class="btn-icon" title="Disconnect">⏻</button>
            </div>
        </header>

        <main id="chat-messages" class="chat-messages">
            <!-- Messages injected here -->
        </main>

        <footer class="chat-input-area">
            <div class="input-actions">
                <input type="file" id="image-upload" accept="image/jpeg,image/png,image/webp" hidden>
                <input type="file" id="gif-upload" accept="image/gif" hidden>
                <input type="file" id="voice-upload" accept="audio/webm,audio/ogg,audio/mpeg" hidden>
                
                <button class="btn-icon" title="Upload Image" onclick="document.getElementById('image-upload').click()">🖼</button>
                <button class="btn-icon" title="Upload GIF" onclick="document.getElementById('gif-upload').click()">🎬</button>
                <button class="btn-icon" title="Upload Voice" onclick="document.getElementById('voice-upload').click()">🎙</button>
            </div>
            
            <form id="message-form" class="message-form">
                <input type="text" id="message-input" placeholder="Type a message..." autocomplete="off" required>
                <button type="submit" class="btn btn-primary">Send</button>
            </form>
        </footer>
    </div>

    <script src="app.js"></script>
</body>
</html>
EOF
cat << 'EOF' > web/static/style.css
/* =========================================
   DESIGN SYSTEM & VARIABLES (Zen Minimalism)
   ========================================= */
:root {
    --bg-primary: #f8f9fa;
    --bg-secondary: #ffffff;
    --text-primary: #212529;
    --text-secondary: #6c757d;
    --accent: #4361ee;
    --accent-hover: #3f37c9;
    --border: #dee2e6;
    --radius-sm: 8px;
    --radius-md: 16px;
    --radius-lg: 24px;
    --shadow-sm: 0 2px 8px rgba(0, 0, 0, 0.03);
    --shadow-md: 0 4px 16px rgba(0, 0, 0, 0.06);
    --transition: 0.2s ease;
    
    /* Chat Bubbles (Gestalt Principle: Similarity) */
    --msg-self: #4361ee;
    --msg-self-text: #ffffff;
    --msg-other: #e9ecef;
    --msg-other-text: #212529;
}

/* =========================================
   BASE RESET & TYPOGRAPHY
   ========================================= */
*, *::before, *::after {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    background-color: var(--bg-primary);
    color: var(--text-primary);
    line-height: 1.5;
    -webkit-font-smoothing: antialiased;
    height: 100vh;
    overflow: hidden;
}

.view {
    height: 100vh;
    display: flex;
    flex-direction: column;
}

/* =========================================
   AUTHENTICATION UI
   ========================================= */
#auth-view {
    justify-content: center;
    align-items: center;
    background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
}

.auth-card {
    background: var(--bg-secondary);
    width: 100%;
    max-width: 420px;
    padding: 3rem;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-md);
    text-align: center;
}

.logo {
    font-size: 2.5rem;
    font-weight: 800;
    color: var(--text-primary);
    margin-bottom: 0.5rem;
    letter-spacing: -1px;
}

.tagline {
    color: var(--text-secondary);
    margin-bottom: 2rem;
    font-size: 1.1rem;
}

.auth-divider {
    display: flex;
    align-items: center;
    margin: 2rem 0;
    color: var(--text-secondary);
    font-size: 0.85rem;
}

.auth-divider::before, .auth-divider::after {
    content: "";
    flex: 1;
    height: 1px;
    background: var(--border);
}

.auth-divider span {
    padding: 0 1rem;
}

.auth-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

input[type="text"], input[type="email"], input[type="password"] {
    width: 100%;
    padding: 1rem 1.25rem;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    font-size: 1rem;
    transition: border-color var(--transition), box-shadow var(--transition);
    background: var(--bg-primary);
}

input:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px rgba(67, 97, 238, 0.15);
}

/* Buttons */
.btn {
    padding: 1rem;
    border: none;
    border-radius: var(--radius-sm);
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: background var(--transition), transform var(--transition);
}

.btn:active { transform: scale(0.98); }

.btn-primary {
    background: var(--accent);
    color: var(--msg-self-text);
}

.btn-primary:hover { background: var(--accent-hover); }

.btn-secondary {
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border);
}

.btn-secondary:hover { background: #e9ecef; }

.btn-full { width: 100%; }

.btn-icon {
    background: none;
    border: none;
    font-size: 1.4rem;
    cursor: pointer;
    padding: 0.5rem;
    border-radius: var(--radius-sm);
    transition: background var(--transition);
    color: var(--text-secondary);
}

.btn-icon:hover { background: var(--bg-primary); }

.link-btn {
    display: block;
    margin-top: 1.5rem;
    color: var(--accent);
    text-decoration: none;
    font-weight: 500;
    font-size: 0.95rem;
}

.link-btn:hover { text-decoration: underline; }

.switch-view { font-size: 0.9rem; color: var(--text-secondary); margin-top: 0.5rem; }
.switch-view a { color: var(--accent); text-decoration: none; font-weight: 500; }
.switch-view a:hover { text-decoration: underline; }

/* =========================================
   CHAT LAYOUT
   ========================================= */
.chat-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 2rem;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border);
    box-shadow: var(--shadow-sm);
    z-index: 10;
}

.room-info h2 { font-size: 1.25rem; }

.status-badge {
    display: inline-block;
    font-size: 0.75rem;
    padding: 0.2rem 0.6rem;
    border-radius: 50px;
    background: #d3f9d8;
    color: #2b8a3e;
    font-weight: 600;
    margin-left: 1rem;
    vertical-align: middle;
}

.user-menu { display: flex; align-items: center; gap: 1rem; }
.username-tag { font-weight: 600; color: var(--text-secondary); font-size: 0.9rem; }

/* Messages Area */
.chat-messages {
    flex: 1;
    overflow-y: auto;
    padding: 2rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    background: var(--bg-primary);
    scroll-behavior: smooth;
}

/* Message Bubbles */
.message {
    max-width: 65%;
    padding: 0.8rem 1.2rem;
    border-radius: var(--radius-md);
    position: relative;
    animation: fadeIn 0.3s ease;
    word-wrap: break-word;
}

@keyframes fadeIn {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
}

.message.self {
    align-self: flex-end;
    background: var(--msg-self);
    color: var(--msg-self-text);
    border-bottom-right-radius: 4px;
}

.message.other {
    align-self: flex-start;
    background: var(--msg-other);
    color: var(--msg-other-text);
    border-bottom-left-radius: 4px;
}

.msg-meta {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    margin-bottom: 0.3rem;
    opacity: 0.8;
}

.msg-sender { font-weight: 700; }
.msg-time { font-style: italic; }

.msg-content { font-size: 1rem; line-height: 1.4; }
.msg-media { max-width: 100%; border-radius: var(--radius-sm); margin-top: 0.5rem; display: block; }

.message.system {
    align-self: center;
    background: none;
    color: var(--text-secondary);
    font-size: 0.85rem;
    padding: 0.5rem;
}

/* Input Area */
.chat-input-area {
    display: flex;
    align-items: center;
    padding: 1rem 2rem;
    background: var(--bg-secondary);
    border-top: 1px solid var(--border);
    gap: 1rem;
}

.input-actions { display: flex; gap: 0.5rem; }

.message-form {
    flex: 1;
    display: flex;
    gap: 1rem;
}

#message-input {
    flex: 1;
    padding: 1rem 1.25rem;
    border: 1px solid var(--border);
    border-radius: 50px;
    font-size: 1rem;
    background: var(--bg-primary);
    transition: border-color var(--transition);
}

#message-input:focus {
    outline: none;
    border-color: var(--accent);
}

.message-form .btn {
    border-radius: 50px;
    padding: 1rem 1.5rem;
}

/* Responsive */
@media (max-width: 768px) {
    .auth-card { margin: 1rem; padding: 2rem; }
    .chat-messages { padding: 1rem; }
    .chat-input-area { padding: 0.5rem 1rem; }
    .message { max-width: 85%; }
    .chat-header { padding: 1rem; }
}
EOF
cat << 'EOF' > web/static/app.js
/**
 * Bare-minimum Vanilla JS for State & WebSocket.
 * Design goal: Keep logic out of the UI layer as much as possible.
 */

// --- State ---
let ws = null;
let token = localStorage.getItem('fc_token');
let username = localStorage.getItem('fc_username');
let currentRoomId = 1; // Defaulting to room 1 for simplicity

// --- DOM Elements ---
const views = {
    auth: document.getElementById('auth-view'),
    chat: document.getElementById('chat-view')
};

const forms = {
    guest: document.getElementById('guest-form'),
    login: document.getElementById('login-form'),
    register: document.getElementById('register-form'),
    message: document.getElementById('message-form')
};

const inputs = {
    guestUsername: document.getElementById('guest-username'),
    loginEmail: document.getElementById('login-email'),
    loginPassword: document.getElementById('login-password'),
    regUsername: document.getElementById('reg-username'),
    regEmail: document.getElementById('reg-email'),
    regPassword: document.getElementById('reg-password'),
    message: document.getElementById('message-input')
};

const chatUI = {
    messages: document.getElementById('chat-messages'),
    displayUsername: document.getElementById('display-username'),
    status: document.getElementById('user-status')
};

// --- Auth View Toggles ---
document.getElementById('show-login').addEventListener('click', (e) => {
    e.preventDefault();
    toggleAuthForms('login');
});
document.getElementById('show-register').addEventListener('click', (e) => {
    e.preventDefault();
    toggleAuthForms('register');
});
document.getElementById('show-login-btn').addEventListener('click', (e) => {
    e.preventDefault();
    toggleAuthForms('login');
});
document.getElementById('show-guest-btn').addEventListener('click', (e) => {
    e.preventDefault();
    toggleAuthForms('guest');
});

function toggleAuthForms(show) {
    forms.guest.style.display = show === 'guest' ? 'flex' : 'none';
    forms.login.style.display = show === 'login' ? 'flex' : 'none';
    forms.register.style.display = show === 'register' ? 'flex' : 'none';
    document.getElementById('show-login-btn').style.display = show === 'guest' ? 'block' : 'none';
    document.getElementById('show-guest-btn').style.display = show !== 'guest' ? 'block' : 'none';
}

// --- API & WebSocket Logic ---
async function handleAuth(form, url, payload) {
    try {
        const res = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        if (!res.ok) {
            const err = await res.text();
            throw new Error(err);
        }

        const data = await res.json();
        
        // Handle different response structures
        if (data.token) {
            setSession(data.token, data.user.username);
        } else if (data.user) {
            // Fallback if token isn't in immediate response (e.g. register)
            // In a real app, you'd redirect to login. For V1, we auto-login guests or show alert.
            alert('Registration successful! Please login.');
            toggleAuthForms('login');
            return;
        }
    } catch (err) {
        alert(err.message);
    }
}

function setSession(t, u) {
    token = t;
    username = u;
    localStorage.setItem('fc_token', t);
    localStorage.setItem('fc_username', u);
    connectWebSocket();
}

function connectWebSocket() {
    // Protocol detection
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    // If token exists, use it. Otherwise, backend allows direct username login via WS
    let wsUrl = `${wsProtocol}//${window.location.host}/ws?`;
    
    if (token) {
        wsUrl += `token=${token}`;
    } else if (username) {
        wsUrl += `username=${encodeURIComponent(username)}`;
    }

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        showChat();
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        handleWSMessage(data);
    };

    ws.onclose = () => {
        chatUI.status.textContent = 'Disconnected';
        chatUI.status.style.background = '#ffe3e3';
        chatUI.status.style.color = '#c92a2a';
    };

    ws.onerror = (err) => {
        console.error('WebSocket Error', err);
        clearSession();
    };
}

function handleWSMessage(data) {
    switch (data.type) {
        case 'auth':
            // Backend sends token back if logged in via direct WS username
            if (data.token) {
                token = data.token;
                localStorage.setItem('fc_token', data.token);
            }
            // Join default room automatically
            sendWS({ type: 'join', room_id: currentRoomId });
            break;
            
        case 'history':
            data.messages.forEach(msg => appendMessage(msg, false));
            break;

        case 'chat':
        case 'media':
            appendMessage(data, true);
            break;

        case 'error':
            console.error('Server Error:', data.error);
            alert(`Error: ${data.error}`);
            break;
    }
}

function sendWS(payload) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(payload));
    }
}

// --- UI Manipulation ---
function showChat() {
    views.auth.style.display = 'none';
    views.chat.style.display = 'flex';
    chatUI.displayUsername.textContent = username;
    inputs.message.focus();
}

function appendMessage(msg, scroll = true) {
    const div = document.createElement('div');
    
    if (msg.type === 'history' && msg.messages) return; // Handled in switch

    const isSelf = msg.sender_id === parseInt(localStorage.getItem('fc_user_id')) || msg.username === username;
    
    div.className = `message ${isSelf ? 'self' : 'other'}`;

    let contentHtml = `<div class="msg-content">${escapeHtml(msg.content || '')}</div>`;
    
    if (msg.media_url) {
        if (msg.media_type === 'voice') {
            contentHtml += `<audio controls class="msg-media" src="${msg.media_url}"></audio>`;
        } else {
            contentHtml += `<img src="${msg.media_url}" class="msg-media" alt="Media" loading="lazy">`;
        }
    }

    const time = msg.timestamp ? new Date(msg.timestamp * 1000).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}) : '';

    div.innerHTML = `
        <div class="msg-meta">
            <span class="msg-sender">${escapeHtml(msg.username)}</span>
            <span class="msg-time">${time}</span>
        </div>
        ${contentHtml}
    `;

    chatUI.messages.appendChild(div);

    if (scroll) {
        chatUI.messages.scrollTop = chatUI.messages.scrollHeight;
    }
}

function clearSession() {
    localStorage.removeItem('fc_token');
    localStorage.removeItem('fc_username');
    token = null;
    username = null;
    views.chat.style.display = 'none';
    views.auth.style.display = 'flex';
    toggleAuthForms('guest');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// --- Media Uploads ---
async function handleMediaUpload(event, endpoint) {
    const file = event.target.files[0];
    if (!file) return;

    const formData = new FormData();
    formData.append(endpoint === '/media/upload/image' ? 'image' : 
                    endpoint === '/media/upload/gif' ? 'gif' : 'voice', file);

    try {
        const res = await fetch(endpoint, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}` },
            body: formData
        });

        if (!res.ok) throw new Error('Upload failed');
        
        const data = await res.json();
        
        // Send media message to websocket
        sendWS({
            type: 'media',
            room_id: currentRoomId,
            media_id: data.id
        });

    } catch (err) {
        alert('Media upload failed: ' + err.message);
    }

    event.target.value = ''; // Reset input
}

document.getElementById('image-upload').addEventListener('change', (e) => handleMediaUpload(e, '/media/upload/image'));
document.getElementById('gif-upload').addEventListener('change', (e) => handleMediaUpload(e, '/media/upload/gif'));
document.getElementById('voice-upload').addEventListener('change', (e) => handleMediaUpload(e, '/media/upload/voice'));

// --- Event Listeners ---
forms.guest.addEventListener('submit', (e) => {
    e.preventDefault();
    const user = inputs.guestUsername.value.trim();
    if (user) connectWebSocket(); // Connects via URL param username
});

forms.login.addEventListener('submit', (e) => {
    e.preventDefault();
    handleAuth(forms.login, '/auth/login', {
        email: inputs.loginEmail.value,
        password: inputs.loginPassword.value
    });
});

forms.register.addEventListener('submit', (e) => {
    e.preventDefault();
    handleAuth(forms.register, '/auth/register', {
        username: inputs.regUsername.value,
        email: inputs.regEmail.value,
        password: inputs.regPassword.value
    });
});

forms.message.addEventListener('submit', (e) => {
    e.preventDefault();
    const content = inputs.message.value.trim();
    if (!content) return;

    sendWS({
        type: 'chat',
        room_id: currentRoomId,
        content: content
    });

    inputs.message.value = '';
    inputs.message.focus();
});

document.getElementById('logout-btn').addEventListener('click', () => {
    if (ws) ws.close();
    clearSession();
});

// --- Init on Load ---
window.onload = () => {
    if (token) {
        connectWebSocket();
    } else {
        toggleAuthForms('guest');
    }
};
EOF
echo "✅ Frontend built successfully in web/static/"
