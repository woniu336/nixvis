// Settings page module

// State
let sites = [];
let scanResults = [];
let excludePatterns = [];
let excludeIPs = [];

// Theme toggle
function initTheme() {
    const savedTheme = localStorage.getItem('theme') || 'light';
    document.documentElement.setAttribute('data-theme', savedTheme);
    updateThemeIcon(savedTheme);
}

function toggleTheme() {
    const currentTheme = document.documentElement.getAttribute('data-theme');
    const newTheme = currentTheme === 'light' ? 'dark' : 'light';
    document.documentElement.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);
    updateThemeIcon(newTheme);
}

function updateThemeIcon(theme) {
    const lightIcon = document.querySelector('.light-icon');
    const darkIcon = document.querySelector('.dark-icon');
    if (theme === 'dark') {
        lightIcon.style.display = 'none';
        darkIcon.style.display = 'inline';
    } else {
        lightIcon.style.display = 'inline';
        darkIcon.style.display = 'none';
    }
}

// API calls
async function fetchSettings() {
    const response = await fetch('/api/settings');
    if (!response.ok) {
        throw new Error('Failed to fetch settings');
    }
    return await response.json();
}

async function scanLogs(path) {
    const response = await fetch('/api/settings/scan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path })
    });
    if (!response.ok) {
        throw new Error('Failed to scan logs');
    }
    return await response.json();
}

async function addSite(name, logPath) {
    const response = await fetch('/api/settings/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, logPath })
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to add site');
    }
    return await response.json();
}

async function removeSite(id) {
    const response = await fetch(`/api/settings/remove/${id}`, {
        method: 'DELETE'
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to remove site');
    }
    return await response.json();
}

async function triggerLogScan() {
    const response = await fetch('/api/settings/scan-logs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to trigger log scan');
    }
    return await response.json();
}

async function addExcludePattern(pattern) {
    const response = await fetch('/api/settings/exclude-patterns', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pattern })
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to add exclude pattern');
    }
    return await response.json();
}

async function removeExcludePattern(pattern) {
    const encodedPattern = encodeURIComponent(pattern);
    const response = await fetch(`/api/settings/exclude-patterns/${encodedPattern}`, {
        method: 'DELETE'
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to remove exclude pattern');
    }
    return await response.json();
}

async function addExcludeIP(ip) {
    const response = await fetch('/api/settings/exclude-ips', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ip })
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to add exclude IP');
    }
    return await response.json();
}

async function removeExcludeIP(ip) {
    const encodedIP = encodeURIComponent(ip);
    const response = await fetch(`/api/settings/exclude-ips/${encodedIP}`, {
        method: 'DELETE'
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to remove exclude IP');
    }
    return await response.json();
}

// Render functions
function renderSitesList() {
    const tbody = document.getElementById('sites-list');
    if (sites.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4">暂无站点</td></tr>';
        return;
    }

    tbody.innerHTML = sites.map(site => `
        <tr>
            <td><code>${escapeHtml(site.id)}</code></td>
            <td>${escapeHtml(site.name)}</td>
            <td><code>${escapeHtml(site.logPath)}</code></td>
            <td>
                <button class="btn-delete" data-id="${escapeHtml(site.id)}" data-name="${escapeHtml(site.name)}">删除</button>
            </td>
        </tr>
    `).join('');

    // Attach delete button handlers
    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', handleDeleteSite);
    });
}

function renderExcludePatterns() {
    const tbody = document.getElementById('exclude-patterns-list');
    if (excludePatterns.length === 0) {
        tbody.innerHTML = '<tr><td colspan="2">暂无排除模式</td></tr>';
        return;
    }

    tbody.innerHTML = excludePatterns.map(pattern => `
        <tr>
            <td><code>${escapeHtml(pattern)}</code></td>
            <td>
                <button class="btn-delete" data-type="pattern" data-value="${escapeHtml(pattern)}">删除</button>
            </td>
        </tr>
    `).join('');

    // Attach delete button handlers
    document.querySelectorAll('#exclude-patterns-list .btn-delete').forEach(btn => {
        btn.addEventListener('click', handleDeleteExcludePattern);
    });
}

function renderExcludeIPs() {
    const tbody = document.getElementById('exclude-ips-list');
    if (excludeIPs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="2">暂无排除IP</td></tr>';
        return;
    }

    tbody.innerHTML = excludeIPs.map(ip => `
        <tr>
            <td><code>${escapeHtml(ip)}</code></td>
            <td>
                <button class="btn-delete" data-type="ip" data-value="${escapeHtml(ip)}">删除</button>
            </td>
        </tr>
    `).join('');

    // Attach delete button handlers
    document.querySelectorAll('#exclude-ips-list .btn-delete').forEach(btn => {
        btn.addEventListener('click', handleDeleteExcludeIP);
    });
}

function renderScanResults() {
    const container = document.getElementById('scan-results');
    const list = document.getElementById('scan-results-list');

    if (scanResults.length === 0) {
        list.innerHTML = '<p class="no-results">未找到可用的日志文件</p>';
    } else {
        list.innerHTML = `
            <div class="scan-results-grid">
                ${scanResults.map(log => `
                    <div class="scan-result-item">
                        <div class="scan-result-name">${escapeHtml(log.name)}</div>
                        <div class="scan-result-path"><code>${escapeHtml(log.path)}</code></div>
                        <button class="btn-add-site" data-path="${escapeHtml(log.path)}" data-name="${escapeHtml(log.name)}">添加为站点</button>
                    </div>
                `).join('')}
            </div>
        `;

        // Attach add site button handlers
        document.querySelectorAll('.btn-add-site').forEach(btn => {
            btn.addEventListener('click', handleAddSiteFromScan);
        });
    }

    container.style.display = 'block';
}

// Event handlers
async function handleScan() {
    const input = document.getElementById('log-path-input');
    const path = input.value.trim();
    const scanBtn = document.getElementById('scan-btn');

    if (!path) {
        alert('请输入日志路径');
        return;
    }

    scanBtn.disabled = true;
    scanBtn.textContent = '扫描中...';

    try {
        const result = await scanLogs(path);
        scanResults = result.logs || [];
        renderScanResults();
    } catch (error) {
        alert('扫描失败: ' + error.message);
    } finally {
        scanBtn.disabled = false;
        scanBtn.textContent = '扫描';
    }
}

function handleAddSiteFromScan(e) {
    const path = e.target.dataset.path;
    let name = e.target.dataset.name;

    // 从日志文件名生成站点名称
    // www.example.com-access.log -> www.example.com
    // www.example.com.log -> www.example.com
    // access.log -> access
    name = name
        .replace(/-access\.log$/, '')
        .replace(/-access_log$/, '')
        .replace(/_access\.log$/, '')
        .replace(/\.access\.log$/, '')
        .replace(/\.log$/, '');

    // 如果名称为空或只是通用名称，使用文件名（不含扩展名）
    if (!name || name === 'access' || name === 'log') {
        name = e.target.dataset.name.replace(/\.[^.]*$/, '');
    }

    document.getElementById('site-log-path').value = path;
    document.getElementById('site-name').value = name;
    showModal();
}

async function handleDeleteSite(e) {
    const id = e.target.dataset.id;
    const name = e.target.dataset.name;

    if (!confirm(`确定要删除站点 "${name}" 吗？`)) {
        return;
    }

    try {
        await removeSite(id);
        await loadSites();
        alert('站点删除成功');
    } catch (error) {
        alert('删除失败: ' + error.message);
    }
}

async function handleConfirmAddSite() {
    const name = document.getElementById('site-name').value.trim();
    const logPath = document.getElementById('site-log-path').value.trim();

    if (!name || !logPath) {
        alert('请填写站点名称和日志路径');
        return;
    }

    try {
        // 添加站点
        await addSite(name, logPath);
        await loadSites();
        hideModal();

        // 显示扫描提示
        alert('站点添加成功！正在扫描日志数据...');

        // 触发日志扫描
        try {
            const scanResult = await triggerLogScan();

            if (scanResult.total_entries > 0) {
                alert(`站点添加成功！已扫描 ${scanResult.total_entries} 条日志记录。`);
            } else {
                alert(`站点添加成功！但未扫描到新的日志记录，请检查日志文件路径是否正确。`);
            }

            // 显示扫描结果详情
            if (scanResult.results && scanResult.results.length > 0) {
                console.log('扫描结果:', scanResult.results);
            }
        } catch (scanError) {
            alert('站点添加成功，但日志扫描失败: ' + scanError.message);
        }
    } catch (error) {
        alert('添加失败: ' + error.message);
    }
}

async function handleAddExcludePattern() {
    const input = document.getElementById('exclude-pattern-input');
    const pattern = input.value.trim();

    if (!pattern) {
        alert('请输入排除模式');
        return;
    }

    try {
        await addExcludePattern(pattern);
        await loadSettings();
        input.value = '';
        alert('排除模式添加成功');
    } catch (error) {
        alert('添加失败: ' + error.message);
    }
}

async function handleDeleteExcludePattern(e) {
    const pattern = e.target.dataset.value;

    if (!confirm(`确定要删除排除模式 "${pattern}" 吗？`)) {
        return;
    }

    try {
        await removeExcludePattern(pattern);
        await loadSettings();
        alert('排除模式删除成功');
    } catch (error) {
        alert('删除失败: ' + error.message);
    }
}

async function handleAddExcludeIP() {
    const input = document.getElementById('exclude-ip-input');
    const ip = input.value.trim();

    if (!ip) {
        alert('请输入IP地址');
        return;
    }

    try {
        await addExcludeIP(ip);
        await loadSettings();
        input.value = '';
        alert('排除IP添加成功');
    } catch (error) {
        alert('添加失败: ' + error.message);
    }
}

async function handleDeleteExcludeIP(e) {
    const ip = e.target.dataset.value;

    if (!confirm(`确定要删除排除IP "${ip}" 吗？`)) {
        return;
    }

    try {
        await removeExcludeIP(ip);
        await loadSettings();
        alert('排除IP删除成功');
    } catch (error) {
        alert('删除失败: ' + error.message);
    }
}

// Modal functions
function showModal() {
    const modal = document.getElementById('add-site-modal');
    modal.classList.add('show');
    modal.style.display = 'flex';
}

function hideModal() {
    const modal = document.getElementById('add-site-modal');
    modal.classList.remove('show');
    modal.style.display = 'none';
    document.getElementById('site-name').value = '';
    document.getElementById('site-log-path').value = '';
}

// Utility functions
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

async function loadSettings() {
    try {
        const data = await fetchSettings();
        sites = data.websites || [];
        excludePatterns = data.excludePatterns || [];
        excludeIPs = data.excludeIPs || [];
        renderSitesList();
        renderExcludePatterns();
        renderExcludeIPs();
    } catch (error) {
        console.error('Failed to load settings:', error);
    }
}

// 保持向后兼容
async function loadSites() {
    await loadSettings();
}

// Password change functionality
async function handleChangePassword(e) {
    e.preventDefault();

    const oldPassword = document.getElementById('old-password').value;
    const newPassword = document.getElementById('new-password').value;
    const newPasswordConfirm = document.getElementById('new-password-confirm').value;
    const changePasswordBtn = document.getElementById('change-password-btn');

    // Validation
    if (!oldPassword || !newPassword || !newPasswordConfirm) {
        alert('请填写所有字段');
        return;
    }

    if (newPassword.length < 6) {
        alert('新密码至少需要6个字符');
        return;
    }

    if (newPassword !== newPasswordConfirm) {
        alert('两次输入的新密码不一致');
        return;
    }

    // Disable button
    changePasswordBtn.disabled = true;
    const originalText = changePasswordBtn.textContent;
    changePasswordBtn.textContent = '提交中...';

    try {
        const response = await fetch('/api/auth/change-password', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                old_password: oldPassword,
                new_password: newPassword
            })
        });

        const data = await response.json();

        if (response.ok) {
            alert('密码修改成功！');
            // Clear form
            document.getElementById('old-password').value = '';
            document.getElementById('new-password').value = '';
            document.getElementById('new-password-confirm').value = '';
        } else {
            alert(data.error || '密码修改失败');
        }
    } catch (error) {
        console.error('Password change error:', error);
        alert('网络错误，请稍后重试');
    } finally {
        changePasswordBtn.disabled = false;
        changePasswordBtn.textContent = originalText;
    }
}

// Password toggle functionality
function initPasswordToggles() {
    const toggleButtons = document.querySelectorAll('.password-toggle');
    toggleButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            const targetId = this.getAttribute('data-target');
            const input = document.getElementById(targetId);
            if (input) {
                if (input.type === 'password') {
                    input.type = 'text';
                    this.textContent = '🙈';
                } else {
                    input.type = 'password';
                    this.textContent = '👁';
                }
            }
        });
    });
}

// Initialize
function init() {
    initTheme();
    loadSettings();

    // Event listeners
    document.getElementById('theme-toggle').addEventListener('click', toggleTheme);
    document.getElementById('scan-btn').addEventListener('click', handleScan);
    document.getElementById('log-path-input').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') handleScan();
    });
    document.getElementById('confirm-add-site').addEventListener('click', handleConfirmAddSite);
    document.getElementById('cancel-add-site').addEventListener('click', hideModal);

    // Exclude pattern event listeners
    document.getElementById('add-exclude-pattern-btn').addEventListener('click', handleAddExcludePattern);
    document.getElementById('exclude-pattern-input').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') handleAddExcludePattern();
    });

    // Exclude IP event listeners
    document.getElementById('add-exclude-ip-btn').addEventListener('click', handleAddExcludeIP);
    document.getElementById('exclude-ip-input').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') handleAddExcludeIP();
    });

    // Password change event listeners
    const changePasswordForm = document.getElementById('change-password-form');
    if (changePasswordForm) {
        changePasswordForm.addEventListener('submit', handleChangePassword);
    }

    // Password toggle buttons
    initPasswordToggles();

    // Modal close on background click
    document.getElementById('add-site-modal').addEventListener('click', (e) => {
        if (e.target.id === 'add-site-modal') hideModal();
    });

    // Modal close on X button
    document.querySelector('.modal-close').addEventListener('click', hideModal);
}

// Start when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
