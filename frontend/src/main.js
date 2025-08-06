import {CheckBackendReady, GetGameDetails, GetGames, TriggerSync} from '../wailsjs/go/main/App';
import {EventsOn} from '../wailsjs/runtime';
import {marked} from 'marked';

const SEARCH_DEBOUNCE_TIME = 300;
const INFINITE_SCROLL_BATCH_SIZE = 20;
const CSS_CLASSES = {
    HIDDEN: 'd-none',
    ACTIVE: 'active',
    DISABLED: 'disabled',
    SPINNER: 'spinner-border',
};
const TRIGGER_ELEMENT_CLASS = 'infinite-scroll-trigger';


const dataCache = new Map();
const detailCache = new Map();

function getCacheKey(searchTerm, limit, offset) {
    return `${searchTerm}|${limit}|${offset}`;
}

function clearCaches() {
    dataCache.clear();
    detailCache.clear();
    console.log('Caches cleared.');
}

const state = new Proxy(
    {
        currentPage: 1,
        pageSize: 20,
        totalLoaded: 0,
        layoutMode: 'infinite',
        viewMode: 'card',
        isLoading: false,
        hasMore: true,
        currentSearch: '',
        isBackendReady: false,
    },
    {
        set(target, property, value) {
            target[property] = value;
            switch (property) {
                case 'isLoading':
                case 'hasMore':
                case 'currentPage':
                    updatePaginationUI();
                    break;
                case 'layoutMode':
                    updateLayout();
                    break;
            }
            return true;
        },
    }
);

const DOMElements = {
    contentArea: document.getElementById('content-area'),
    searchInput: document.getElementById('searchInput'),
    syncStatusEl: document.getElementById('syncStatus'),
    refreshButton: document.getElementById('refresh-button'),
    layoutInfinite: document.getElementById('layout-infinite'),
    layoutPagination: document.getElementById('layout-pagination'),
    resultsInfinite: document.getElementById('results-infinite'),
    resultsPagination: document.getElementById('results-pagination'),
    prevPageButton: document.getElementById('prev-page-button'),
    nextPageButton: document.getElementById('next-page-button'),
    pageInfo: document.getElementById('page-info'),
    prevPageItem: document.getElementById('prev-page-item'),
    nextPageItem: document.getElementById('next-page-item'),
    mainLayout: document.getElementById('main-layout'),
    detailView: document.getElementById('detail-view'),
    detailBackButton: document.getElementById('detail-back-button'),
    scrollToTopButton: document.getElementById('scroll-to-top'),
    pageSizeOptions: document.querySelectorAll('.page-size-option'),
    layoutModeOptions: document.querySelectorAll('.layout-mode-option'),
    viewModeOptions: document.querySelectorAll('.view-mode-option'),
    themeLight: document.getElementById('theme-light'),
    themeDark: document.getElementById('theme-dark'),
    imageViewerModalEl: document.getElementById('image-viewer-modal'),
    imageViewerSrc: document.getElementById('image-viewer-src'),
};

const imageViewerModal = new bootstrap.Modal(DOMElements.imageViewerModalEl);

const observer = new IntersectionObserver(
    (entries) => {
        if (entries[0]?.isIntersecting && !state.isLoading && state.hasMore) {
            loadGames(state.currentPage + 1);
        }
    },
    {threshold: 0.1}
);

function showSkeletons(container, count) {
    container.innerHTML = '';
    const fragment = document.createDocumentFragment();
    for (let i = 0; i < count; i++) {
        const skeleton = document.createElement('div');
        if (state.viewMode === 'card') {
            skeleton.className = 'game-card card shadow-sm skeleton';
            skeleton.innerHTML = `
                <div class="skeleton-img"></div>
                <div class="card-body">
                    <div class="skeleton-text w-75"></div>
                    <div class="skeleton-text w-50"></div>
                </div>`;
        } else {
            skeleton.className = 'game-list-item skeleton';
            skeleton.innerHTML = `
                <div class="skeleton-img-list"></div>
                <div class="w-100">
                    <div class="skeleton-text w-75"></div>
                    <div class="skeleton-text w-50"></div>
                </div>
                <div class="skeleton-text w-25"></div>`;
        }
        fragment.appendChild(skeleton);
    }
    container.appendChild(fragment);
}

function createGameElement(game) {
    const title = game.TitleCN || game.TitleJP;
    const cover_url = game.CoverURL || 'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7';

    const div = document.createElement('div');
    div.dataset.id = game.ID;
    div.dataset.title = title;
    div.dataset.brand = game.Brand || '未知';
    div.dataset.release = game.ReleaseDate || '未知';
    div.dataset.cover = cover_url;

    renderGameElementContent(div, state.viewMode);
    return div;
}

function renderGameElementContent(element, viewMode) {
    const {title, brand, release, cover} = element.dataset;
    element.className = viewMode === 'card' ? 'game-card card shadow-sm' : 'game-list-item';
    element.innerHTML = viewMode === 'card'
        ? `
            <img src="${cover}" class="card-img-top" alt="${title}" loading="lazy" draggable="false" onerror="this.src='${cover}';">
            <div class="card-body">
                <h6 class="card-title text-truncate fw-bold">${title}</h6>
                <p class="card-text text-muted small text-truncate">${brand} · ${release}</p>
            </div>
        `
        : `
            <img src="${cover}" class="list-cover" alt="${title}" loading="lazy" draggable="false" onerror="this.src='${cover}';">
            <div>
                <strong class="text-truncate d-block">${title}</strong>
                <small class="text-muted">${brand}</small>
            </div>
            <span class="text-muted text-end small">${release}</span>
        `;
}

function displayResults(container, games) {
    if (!Array.isArray(games) || games.length === 0) {
        if (container.childElementCount === 0) {
            container.innerHTML = '<p class="text-center text-muted w-100 p-4">未找到结果。</p>';
        }
        return;
    }
    const fragment = document.createDocumentFragment();
    games.forEach((game) => fragment.appendChild(createGameElement(game)));
    container.appendChild(fragment);
}

function updateLayout() {
    DOMElements.pageSizeOptions.forEach((el) =>
        el.classList.toggle(CSS_CLASSES.ACTIVE, parseInt(el.dataset.size) === state.pageSize)
    );
    DOMElements.layoutModeOptions.forEach((el) =>
        el.classList.toggle(CSS_CLASSES.ACTIVE, el.dataset.mode === state.layoutMode)
    );

    const isInfinite = state.layoutMode === 'infinite';
    DOMElements.layoutInfinite.classList.toggle(CSS_CLASSES.HIDDEN, !isInfinite);
    DOMElements.layoutPagination.classList.toggle(CSS_CLASSES.HIDDEN, isInfinite);

    observer.disconnect();

    if (isInfinite) {
        const existingTrigger = DOMElements.resultsInfinite.querySelector(`.${TRIGGER_ELEMENT_CLASS}`);
        if (existingTrigger) observer.observe(existingTrigger);
    }
}

function applyViewMode(newMode) {
    if (state.viewMode === newMode) return;
    state.viewMode = newMode;
    localStorage.setItem('viewMode', newMode);

    DOMElements.viewModeOptions.forEach((el) =>
        el.classList.toggle(CSS_CLASSES.ACTIVE, el.dataset.view === newMode)
    );

    document.querySelectorAll('.results-view').forEach(container => {
        container.dataset.viewMode = newMode;
        container.querySelectorAll('[data-id]').forEach(item => {
            renderGameElementContent(item, newMode);
        });
    });
}

function setTheme(theme) {
    document.documentElement.setAttribute('data-bs-theme', theme);
    localStorage.setItem('theme', theme);
}

function updatePaginationUI() {
    if (state.layoutMode !== 'pagination') return;
    DOMElements.pageInfo.textContent = `第 ${state.currentPage} 页`;
    DOMElements.prevPageItem.classList.toggle(CSS_CLASSES.DISABLED, state.isLoading || state.currentPage <= 1);
    DOMElements.nextPageItem.classList.toggle(CSS_CLASSES.DISABLED, state.isLoading || !state.hasMore);
}

function updateInfiniteScrollTrigger() {
    const container = DOMElements.resultsInfinite;

    const oldTrigger = container.querySelector(`.${TRIGGER_ELEMENT_CLASS}`);
    if (oldTrigger) oldTrigger.remove();

    if (state.layoutMode !== 'infinite') return;

    const triggerEl = document.createElement('div');
    triggerEl.className = TRIGGER_ELEMENT_CLASS;
    triggerEl.style.gridColumn = '1 / -1';
    triggerEl.style.width = '100%';

    if (state.isLoading && state.currentPage > 1) {
        triggerEl.innerHTML = `<div class="d-flex justify-content-center p-3 w-100"><div class="${CSS_CLASSES.SPINNER} text-primary"></div></div>`;
        container.appendChild(triggerEl);
    } else if (state.hasMore) {
        container.appendChild(triggerEl);
        observer.observe(triggerEl);
    } else if (!state.hasMore && state.totalLoaded > 0) {
        triggerEl.innerHTML = `<p class="text-muted text-center p-3 w-100">已加载全部数据</p>`;
        container.appendChild(triggerEl);
    }
}

async function loadGames(page = 1) {
    if (state.isLoading || !state.isBackendReady) return;

    state.isLoading = true;
    state.currentPage = page;

    observer.disconnect();

    const isInfinite = state.layoutMode === 'infinite';
    const targetContainer = isInfinite ? DOMElements.resultsInfinite : DOMElements.resultsPagination;
    const isFirstPage = page === 1;

    const shouldReplaceContent = !isInfinite || isFirstPage;

    const batchSize = isInfinite ? INFINITE_SCROLL_BATCH_SIZE : state.pageSize;
    const offset = isInfinite ? (isFirstPage ? 0 : state.totalLoaded) : (page - 1) * state.pageSize;
    const cacheKey = getCacheKey(state.currentSearch, batchSize, offset);

    try {
        if (shouldReplaceContent) {
            showSkeletons(targetContainer, batchSize);
        }

        let games;
        if (dataCache.has(cacheKey)) {
            games = dataCache.get(cacheKey);
        } else {
            games = await GetGames(state.currentSearch, batchSize, offset);
            if (Array.isArray(games)) dataCache.set(cacheKey, games);
        }

        if (shouldReplaceContent) {
            targetContainer.innerHTML = '';
        }

        displayResults(targetContainer, games);

        state.hasMore = Array.isArray(games) && games.length === batchSize;
        if (isInfinite) {
            if (isFirstPage) state.totalLoaded = 0;
            if (Array.isArray(games)) state.totalLoaded += games.length;
        }

    } catch (error) {
        console.error('加载游戏失败:', error);
        if (shouldReplaceContent) {
            targetContainer.innerHTML = `<p class="text-center text-danger w-100 p-4">加载失败，请重试: ${error.message}</p>`;
        }
        state.hasMore = false;
    } finally {
        state.isLoading = false;
        if (isInfinite) {
            updateInfiniteScrollTrigger();
        }
    }
}

async function showDetailView(id) {
    DOMElements.mainLayout.classList.add(CSS_CLASSES.HIDDEN);
    DOMElements.detailView.classList.remove(CSS_CLASSES.HIDDEN);
    const content = DOMElements.detailView.querySelector('.detail-content');
    content.scrollTop = 0;
    content.innerHTML = `<div class="d-flex justify-content-center p-5"><div class="${CSS_CLASSES.SPINNER}" style="width: 3rem; height: 3rem;"></div></div>`;
    try {
        let game;
        if (detailCache.has(id)) {
            game = detailCache.get(id);
        } else {
            game = await GetGameDetails(id);
            detailCache.set(id, game);
        }
        content.innerHTML = renderGameDetails(game);
    } catch (error) {
        console.error('加载详情失败:', error);
        detailCache.delete(id);
        content.innerHTML = `<p class="text-center text-danger p-5">加载详情失败，请重试: ${error.message}</p>`;
    }
}

function hideDetailView() {
    DOMElements.detailView.classList.add(CSS_CLASSES.HIDDEN);
    DOMElements.mainLayout.classList.remove(CSS_CLASSES.HIDDEN);
}

function renderGameDetails(game) {
    const releaseDate = game.release_date ? new Date(game.release_date).toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: 'long',
        day: 'numeric'
    }) : '未知';
    const displayTitle = game.title_cn || game.title_jp || '无标题';
    const originalTitle = game.title_cn && game.title_jp ? `<h3 class="text-muted fw-light mb-4">${game.title_jp}</h3>` : '';
    const coverUrl = game.cover_url || 'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7';
    return `<div class="container-fluid"><div class="row g-4 py-4"><div class="col-lg-4"><div class="detail-cover-container"><img src="${coverUrl}" class="detail-cover mb-4" onerror="this.style.display='none'" draggable="false">${renderDetailInfoCard(game.brand, releaseDate)}</div></div><div class="col-lg-8"><div class="p-lg-3"><h1 class="display-6 fw-bold mb-1">${displayTitle}</h1>${originalTitle}<div id="detail-tags-container" class="mb-4 d-flex flex-wrap gap-2">${renderTags(game.tags)}</div><hr class="my-4">${renderSynopsis(game.synopsis)}${renderPreviews(game.preview_urls)}${renderDownloads(game.download_link)}</div></div></div></div>`;
}

function renderDetailInfoCard(brand, releaseDate) {
    return `<div class="card info-card"><div class="card-body p-4"><h5 class="card-title mb-3">游戏信息</h5><ul class="list-unstyled mb-0"><li class="mb-2 d-flex align-items-center"><i class="bi bi-building fs-5 me-3 text-muted"></i><strong>${brand || '未知'}</strong></li><li class="d-flex align-items-center"><i class="bi bi-calendar-event fs-5 me-3 text-muted"></i><span>${releaseDate}</span></li></ul></div></div>`;
}

function renderTags(tags) {
    if (!tags || !tags.trim()) return '<p class="text-muted mb-0">暂无标签</p>';
    return tags.split(',').map(tag => {
        const trimmed = tag.trim();
        return trimmed ? `<a href="#" class="badge rounded-pill tag-badge text-decoration-none" data-tag="${trimmed}">${trimmed}</a>` : '';
    }).filter(Boolean).join('');
}

function renderSynopsis(synopsis) {
    const content = synopsis ? marked.parse(synopsis) : '<p class="text-muted">暂无简介。</p>';
    return `<div class="card info-card mb-4"><div class="card-body"><h5 class="card-title">游戏简介</h5><div id="synopsis-content" class="synopsis-content collapsed text-break mt-3">${content}</div><div class="text-center mt-3"><a href="#" data-action="toggle-synopsis" class="synopsis-toggle btn btn-sm btn-outline-primary rounded-pill px-3"><i class="bi bi-chevron-down me-1"></i>展开阅读</a></div></div></div>`;
}

function renderPreviews(previews) {
    const content = previews && previews.trim() ? previews.split(',').map(url => {
        const trimmed = url.trim();
        return trimmed ? `<img src="${trimmed}" alt="预览图" loading="lazy" data-action="view-image" data-url="${trimmed}" draggable="false">` : '';
    }).filter(Boolean).join('') : '<p class="text-muted">暂无预览图</p>';
    return `<div class="card info-card mb-4"><div class="card-body"><h5 class="card-title">预览图</h5><div class="preview-gallery mt-3">${content}</div></div></div>`;
}

function parseAndRenderDownloads(downloadLinkText) {
    if (!downloadLinkText) return '<p class="text-muted">暂无资源链接。</p>';
    let links = [];
    try {
        links = JSON.parse(downloadLinkText);
    } catch {
        links = [{type: '原始链接', url: downloadLinkText, status: 'valid'}];
    }
    if (!Array.isArray(links) || links.length === 0) {
        return '<p class="text-muted">暂无资源链接。</p>';
    }
    return links.map(link => {
        const statusIcon = {
            valid: '<i class="bi bi-check-circle-fill text-success"></i>',
            warning: '<i class="bi bi-exclamation-triangle-fill text-warning"></i>',
            invalid: '<i class="bi bi-x-circle-fill text-danger"></i>',
        }[link.status] || '';
        const linkUrl = link.url || '#';
        const infoParts = [link.size, link.platform, link.language, link.host].filter(Boolean);
        const infoHTML = infoParts.length ? `<small class="text-muted">${infoParts.join(' · ')}</small>` : '';
        return `<div class="resource-item"><div class="resource-item-info"><strong class="me-2">${link.type || 'N/A'}</strong>${infoHTML}</div><div class="resource-item-actions"><button class="btn btn-sm btn-outline-secondary" data-action="copy-link" data-link="${linkUrl}"><i class="bi bi-clipboard"></i> 复制</button><a href="${linkUrl}" target="_blank" rel="noopener noreferrer" class="btn btn-sm btn-primary"><i class="bi bi-box-arrow-up-right"></i> 打开</a><span class="resource-item-status fs-5 d-flex align-items-center ms-2">${statusIcon}</span></div></div>`;
    }).join('');
}

function renderDownloads(downloadLinkText) {
    const content = parseAndRenderDownloads(downloadLinkText);
    return `<div class="card info-card"><div class="card-body"><h5 class="card-title">资源链接</h5><div id="detail-download-container" class="mt-3">${content}</div></div></div>`;
}

function toggleSynopsis(button) {
    const content = document.getElementById('synopsis-content');
    const isCollapsed = content.classList.toggle('collapsed');
    button.innerHTML = isCollapsed ? '<i class="bi bi-chevron-down me-1"></i>展开阅读' : '<i class="bi bi-chevron-up me-1"></i>收起';
}

function copyToClipboard(text, element) {
    if (!text || text === '#') return;
    navigator.clipboard.writeText(text).then(() => {
        const originalHTML = element.innerHTML;
        element.innerHTML = '<i class="bi bi-check-lg"></i> 已复制';
        element.classList.replace('btn-outline-secondary', 'btn-success');
        setTimeout(() => {
            element.innerHTML = originalHTML;
            element.classList.replace('btn-success', 'btn-outline-secondary');
        }, 2000);
    }).catch(err => console.error('复制失败:', err));
}

function setupEventListeners() {
    let searchTimeout;
    DOMElements.searchInput.addEventListener('input', () => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => startNewSearch(DOMElements.searchInput.value), SEARCH_DEBOUNCE_TIME);
    });
    DOMElements.refreshButton.addEventListener('click', () => {
        DOMElements.searchInput.value = '';
        state.currentSearch = '';
        TriggerSync();
    });
    DOMElements.mainLayout.addEventListener('click', (e) => {
        const target = e.target.closest('.game-card, .game-list-item');
        if (target?.dataset.id) {
            showDetailView(parseInt(target.dataset.id));
        }
    });
    DOMElements.detailView.addEventListener('click', (e) => {
        const tagBadge = e.target.closest('.tag-badge[data-tag]');
        if (tagBadge) {
            e.preventDefault();
            hideDetailView();
            startNewSearch(`tag:${tagBadge.dataset.tag}`);
            return;
        }
        const actionTarget = e.target.closest('[data-action]');
        if (!actionTarget) return;
        e.preventDefault();
        switch (actionTarget.dataset.action) {
            case 'toggle-synopsis':
                toggleSynopsis(actionTarget);
                break;
            case 'view-image':
                DOMElements.imageViewerSrc.src = actionTarget.dataset.url;
                imageViewerModal.show();
                break;
            case 'copy-link':
                copyToClipboard(actionTarget.dataset.link, actionTarget);
                break;
        }
    });
    DOMElements.detailBackButton.addEventListener('click', hideDetailView);
    DOMElements.contentArea.addEventListener('scroll', () => {
        DOMElements.scrollToTopButton.style.display = DOMElements.contentArea.scrollTop > 300 ? 'block' : 'none';
    });
    DOMElements.scrollToTopButton.addEventListener('click', () => DOMElements.contentArea.scrollTo({
        top: 0,
        behavior: 'smooth'
    }));
    DOMElements.prevPageButton.addEventListener('click', (e) => {
        e.preventDefault();
        if (!DOMElements.prevPageItem.classList.contains(CSS_CLASSES.DISABLED)) {
            loadGames(state.currentPage - 1);
        }
    });
    DOMElements.nextPageButton.addEventListener('click', (e) => {
        e.preventDefault();
        if (!DOMElements.nextPageItem.classList.contains(CSS_CLASSES.DISABLED)) {
            loadGames(state.currentPage + 1);
        }
    });
    DOMElements.pageSizeOptions.forEach(el => el.addEventListener('click', () => {
        state.pageSize = parseInt(el.dataset.size);
        localStorage.setItem('pageSize', state.pageSize);
        startNewSearch(state.currentSearch, true);
    }));
    DOMElements.layoutModeOptions.forEach(el => el.addEventListener('click', () => {
        state.layoutMode = el.dataset.mode;
        localStorage.setItem('layoutMode', state.layoutMode);
        startNewSearch(state.currentSearch, true);
    }));
    DOMElements.viewModeOptions.forEach(el => el.addEventListener('click', () => {
        applyViewMode(el.dataset.view);
    }));
    DOMElements.themeLight.addEventListener('click', () => setTheme('light'));
    DOMElements.themeDark.addEventListener('click', () => setTheme('dark'));
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && !DOMElements.detailView.classList.contains(CSS_CLASSES.HIDDEN)) {
            hideDetailView();
        }
    });
    EventsOn('sync-status', (status) => {
        DOMElements.syncStatusEl.style.display = 'block';
        DOMElements.syncStatusEl.textContent = status;
        DOMElements.searchInput.disabled = !status.includes('完成');
        if (status.includes('完成')) {
            clearCaches();
            startNewSearch('', true);
            setTimeout(() => (DOMElements.syncStatusEl.style.display = 'none'), 5000);
        }
    });
}

function startNewSearch(keyword, force = false) {
    const newSearchTerm = keyword.trim();
    if (newSearchTerm === state.currentSearch && !force) return;
    state.currentSearch = newSearchTerm;
    if (document.activeElement !== DOMElements.searchInput) {
        DOMElements.searchInput.value = newSearchTerm;
    }
    state.currentPage = 1;
    state.totalLoaded = 0;
    state.hasMore = true;
    updateLayout();
    loadGames(1);
    DOMElements.contentArea.scrollTo({top: 0, behavior: 'smooth'});
}

function applyInitialSettings() {
    state.pageSize = parseInt(localStorage.getItem('pageSize')) || 20;
    state.layoutMode = localStorage.getItem('layoutMode') || 'infinite';
    const savedViewMode = localStorage.getItem('viewMode') || 'card';
    state.viewMode = savedViewMode;
    DOMElements.viewModeOptions.forEach((el) => el.classList.toggle(CSS_CLASSES.ACTIVE, el.dataset.view === savedViewMode));
    document.querySelectorAll('.results-view').forEach((container) => (container.dataset.viewMode = savedViewMode));
    setTheme(localStorage.getItem('theme') || 'light');
    updateLayout();
}

document.addEventListener('DOMContentLoaded', async () => {
    applyInitialSettings();
    setupEventListeners();
    try {
        const ready = await CheckBackendReady();
        if (ready) {
            state.isBackendReady = true;
            loadGames(1);
        } else {
            EventsOn('backend-ready', () => {
                if (state.isBackendReady) return;
                state.isBackendReady = true;
                loadGames(1);
            });
        }
    } catch (error) {
        console.error('检查后端状态失败:', error);
    }
});