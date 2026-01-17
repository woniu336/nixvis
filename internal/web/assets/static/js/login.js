// Login page logic
import { isAuthenticated, checkAuthStatus, getAuthToken } from './auth.js';

// DOM elements
let loginForm, initForm;
let loginBtn, initBtn;
let usernameInput, passwordInput;
let initUsernameInput, initPasswordInput, initPasswordConfirmInput;
let alertContainer;
let passwordToggleButtons;

// Check authentication status on page load
async function init() {
    // Get DOM elements
    loginForm = document.getElementById('login-form');
    initForm = document.getElementById('init-form');
    loginBtn = document.getElementById('login-btn');
    initBtn = document.getElementById('init-btn');
    usernameInput = document.getElementById('username');
    passwordInput = document.getElementById('password');
    initUsernameInput = document.getElementById('init-username');
    initPasswordInput = document.getElementById('init-password');
    initPasswordConfirmInput = document.getElementById('init-password-confirm');
    alertContainer = document.getElementById('alert-container');
    passwordToggleButtons = document.querySelectorAll('.password-toggle');

    // Setup password toggle
    passwordToggleButtons.forEach(btn => {
        btn.addEventListener('click', togglePassword);
    });

    // Check if already authenticated
    const authStatus = await checkAuthStatus();
    if (authStatus && authStatus.authenticated) {
        // Already logged in, redirect to home
        window.location.href = '/';
        return;
    }

    // Check if system is initialized
    try {
        const response = await fetch('/api/auth/check');
        if (response.ok) {
            const data = await response.json();
            if (!data.initialized) {
                // Show init form
                showInitForm();
            } else {
                // Show login form
                showLoginForm();
            }
        } else {
            showAlert('检查系统状态失败', 'error');
            showLoginForm();
        }
    } catch (error) {
        console.error('Error checking auth status:', error);
        showAlert('连接服务器失败', 'error');
        showLoginForm();
    }

    // Setup form submissions
    if (loginForm) {
        loginForm.addEventListener('submit', handleLogin);
    }
    if (initForm) {
        initForm.addEventListener('submit', handleInit);
    }
}

function showLoginForm() {
    if (loginForm) loginForm.style.display = 'block';
    if (initForm) initForm.style.display = 'none';
}

function showInitForm() {
    if (loginForm) loginForm.style.display = 'none';
    if (initForm) initForm.style.display = 'block';
}

function togglePassword(e) {
    const button = e.target;
    const input = button.parentElement.querySelector('input');

    if (input.type === 'password') {
        input.type = 'text';
        button.textContent = '🙈';
    } else {
        input.type = 'password';
        button.textContent = '👁';
    }
}

async function handleLogin(e) {
    e.preventDefault();

    const username = usernameInput.value.trim();
    const password = passwordInput.value;

    if (!username || !password) {
        showAlert('请输入用户名和密码', 'error');
        return;
    }

    setLoading(loginBtn, true);

    try {
        const response = await fetch('/api/auth/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password }),
        });

        const data = await response.json();

        if (response.ok) {
            showAlert('登录成功，正在跳转...', 'success');
            // Store token in localStorage as backup
            if (data.token) {
                localStorage.setItem('auth_token', data.token);
            }
            // Redirect to home after short delay
            setTimeout(() => {
                window.location.href = '/';
            }, 500);
        } else {
            showAlert(data.error || '登录失败', 'error');
            setLoading(loginBtn, false);
        }
    } catch (error) {
        console.error('Login error:', error);
        showAlert('网络错误，请稍后重试', 'error');
        setLoading(loginBtn, false);
    }
}

async function handleInit(e) {
    e.preventDefault();

    const username = initUsernameInput.value.trim();
    const password = initPasswordInput.value;
    const passwordConfirm = initPasswordConfirmInput.value;

    if (!username) {
        showAlert('请输入管理员用户名', 'error');
        return;
    }

    if (password.length < 6) {
        showAlert('密码至少需要6个字符', 'error');
        return;
    }

    if (password !== passwordConfirm) {
        showAlert('两次输入的密码不一致', 'error');
        return;
    }

    setLoading(initBtn, true);

    try {
        const response = await fetch('/api/auth/initialize', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password }),
        });

        const data = await response.json();

        if (response.ok) {
            showAlert('管理员账户创建成功，请登录', 'success');
            // Switch to login form and pre-fill username
            setTimeout(() => {
                showLoginForm();
                usernameInput.value = username;
                setLoading(initBtn, false);
            }, 1000);
        } else {
            showAlert(data.error || '创建失败', 'error');
            setLoading(initBtn, false);
        }
    } catch (error) {
        console.error('Init error:', error);
        showAlert('网络错误，请稍后重试', 'error');
        setLoading(initBtn, false);
    }
}

function showAlert(message, type = 'error') {
    if (!alertContainer) return;

    // Clear existing alerts
    alertContainer.innerHTML = '';

    const alert = document.createElement('div');
    alert.className = `alert alert-${type}`;
    alert.textContent = message;

    alertContainer.appendChild(alert);

    // Auto-hide success alerts after 3 seconds
    if (type === 'success') {
        setTimeout(() => {
            if (alertContainer.contains(alert)) {
                alertContainer.removeChild(alert);
            }
        }, 3000);
    }
}

function setLoading(button, loading) {
    if (!button) return;

    if (loading) {
        button.disabled = true;
        const originalText = button.textContent;
        button.setAttribute('data-original-text', originalText);
        button.innerHTML = '<span class="loading-spinner"></span>加载中...';
    } else {
        button.disabled = false;
        const originalText = button.getAttribute('data-original-text');
        if (originalText) {
            button.textContent = originalText;
        }
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', init);
