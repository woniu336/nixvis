// Authentication utilities for NixVis

// Get auth token from cookie or localStorage
export function getAuthToken() {
    // First try to get from cookie
    const cookies = document.cookie.split(';');
    for (let cookie of cookies) {
        const [name, value] = cookie.trim().split('=');
        if (name === 'auth_token') {
            return value;
        }
    }

    // Fallback to localStorage
    return localStorage.getItem('auth_token');
}

// Check if user is authenticated
export async function isAuthenticated() {
    const token = getAuthToken();
    if (!token) {
        return false;
    }

    try {
        const response = await fetch('/api/auth/me', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        return response.ok;
    } catch (error) {
        console.error('Auth check error:', error);
        return false;
    }
}

// Get current user info
export async function getCurrentUser() {
    try {
        const response = await fetch('/api/auth/me');
        if (response.ok) {
            return await response.json();
        }
    } catch (error) {
        console.error('Get user error:', error);
    }
    return null;
}

// Check auth status and return user info if authenticated
export async function checkAuthStatus() {
    try {
        const response = await fetch('/api/auth/me');
        if (response.ok) {
            return await response.json();
        }
    } catch (error) {
        console.error('Check auth status error:', error);
    }
    return null;
}

// Logout
export async function logout() {
    try {
        await fetch('/api/auth/logout', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        });
    } catch (error) {
        console.error('Logout error:', error);
    }

    // Clear local storage
    localStorage.removeItem('auth_token');

    // Redirect to login page
    window.location.href = '/login';
}

// Change password
export async function changePassword(oldPassword, newPassword) {
    const response = await fetch('/api/auth/change-password', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            old_password: oldPassword,
            new_password: newPassword,
        }),
    });

    const data = await response.json();

    if (response.ok) {
        return { success: true, message: data.message };
    } else {
        return { success: false, error: data.error };
    }
}

// Setup auth redirect for protected pages
// This function checks if user is authenticated and redirects to login if not
export async function requireAuth() {
    const authStatus = await checkAuthStatus();
    if (!authStatus || !authStatus.authenticated) {
        // Store current path for redirect after login
        sessionStorage.setItem('redirect_after_login', window.location.pathname);
        window.location.href = '/login';
        return false;
    }
    return true;
}

// Auto-redirect to intended page after login
export function getRedirectPath() {
    const redirectPath = sessionStorage.getItem('redirect_after_login');
    sessionStorage.removeItem('redirect_after_login');
    return redirectPath || '/';
}

// Add auth header to fetch options
export function withAuth(options = {}) {
    const token = getAuthToken();
    if (!token) {
        return options;
    }

    options.headers = options.headers || {};

    // If headers is a Headers object, use append
    if (options.headers instanceof Headers) {
        options.headers.append('Authorization', `Bearer ${token}`);
    } else {
        options.headers['Authorization'] = `Bearer ${token}`;
    }

    return options;
}

// Wrapper for fetch with auth
export async function authFetch(url, options = {}) {
    const authOptions = withAuth(options);
    const response = await fetch(url, authOptions);

    // Handle 401 Unauthorized - redirect to login
    if (response.status === 401 && window.location.pathname !== '/login') {
        sessionStorage.setItem('redirect_after_login', window.location.pathname);
        window.location.href = '/login';
    }

    return response;
}

// Initialize auth UI (add logout button, user info, etc.)
export function initAuthUI() {
    checkAuthStatus().then(authStatus => {
        if (authStatus && authStatus.authenticated) {
            // Add user menu or logout button to header
            addAuthUI(authStatus.username);
        }
    });
}

function addAuthUI(username) {
    // Find header actions container
    const headerActions = document.querySelector('.header-actions');
    if (!headerActions) return;

    // Create user info element
    const userInfo = document.createElement('div');
    userInfo.className = 'user-info';
    userInfo.style.cssText = 'display: flex; align-items: center; gap: 12px;';

    // Create username display
    const usernameSpan = document.createElement('span');
    usernameSpan.className = 'username-display';
    usernameSpan.textContent = username;
    usernameSpan.style.cssText = 'font-size: 14px; color: var(--text-color);';

    // Create logout button
    const logoutBtn = document.createElement('button');
    logoutBtn.className = 'nav-button';
    logoutBtn.textContent = '退出登录';
    logoutBtn.onclick = logout;

    userInfo.appendChild(usernameSpan);
    userInfo.appendChild(logoutBtn);

    // Insert before the theme toggle button
    const themeToggle = headerActions.querySelector('.theme-toggle');
    if (themeToggle) {
        headerActions.insertBefore(userInfo, themeToggle);
    } else {
        headerActions.appendChild(userInfo);
    }
}

// Auto-initialize on page load for all pages except login
if (window.location.pathname !== '/login') {
    document.addEventListener('DOMContentLoaded', () => {
        // Check auth status for protected pages
        const protectedPaths = ['/logs', '/spiders', '/suspicious', '/settings'];
        if (protectedPaths.some(path => window.location.pathname.startsWith(path))) {
            requireAuth();
        }
        initAuthUI();
    });
}
