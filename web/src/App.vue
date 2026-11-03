<template>
  <div class="layout">
    <aside class="sidebar">
      <RouterLink class="brand" to="/" aria-label="Wordweave 生词本首页">
        <span class="brand-mark" aria-hidden="true">W<span>·</span></span>
        <span>Wordweave<small>我的生词本</small></span>
      </RouterLink>
      <div class="nav-label">我的学习空间</div>
      <nav aria-label="主要导航">
        <RouterLink class="nav-link" to="/" exact-active-class="selected">
          <span aria-hidden="true">▤</span> 全部词条
        </RouterLink>
        <RouterLink class="nav-link" to="/entries/new" active-class="selected">
          <span aria-hidden="true">＋</span> 收录词条
        </RouterLink>
      </nav>
      <div class="sidebar-note">
        <span class="status-dot"></span> 本机生词本<small
          >一点一滴，织成自己的词汇。</small
        >
      </div>
    </aside>
    <div class="workspace">
      <header class="topbar">
        <span>个人词汇收藏</span><span class="local-badge">专注每一次积累</span>
      </header>
      <main id="main"><RouterView :key="$route.fullPath" /></main>
      <footer>WORDWEAVE <span>让遇见的词，成为自己的语言。</span></footer>
    </div>
  </div>
</template>
