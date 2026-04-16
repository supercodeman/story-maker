<!-- web/src/views/market/MemoryMarket.vue -->
<template>
  <div class="memory-market">
    <div class="page-header">
      <div>
        <h1 class="page-title">记忆市场</h1>
        <p class="page-desc">发现和购买优质写作记忆，提升你的创作能力</p>
      </div>
    </div>

    <!-- 筛选栏 -->
    <FadePanel class="filter-panel">
      <div class="filter-row">
        <div class="filter-tabs">
          <button
            v-for="tab in categoryTabs"
            :key="tab.value"
            class="filter-tab"
            :class="{ 'filter-tab--active': filterCategory === tab.value }"
            @click="filterCategory = tab.value; loadMemories()"
          >
            <span class="filter-tab__icon">{{ tab.icon }}</span>
            <span class="filter-tab__label">{{ tab.label }}</span>
          </button>
        </div>
        <div class="filter-actions">
          <div class="search-box">
            <span class="search-box__icon">🔍</span>
            <input
              v-model="keyword"
              class="search-box__input"
              placeholder="搜索记忆..."
              @keyup.enter="loadMemories"
            />
          </div>
          <div class="sort-tabs">
            <button
              v-for="s in sortOptions"
              :key="s.value"
              class="sort-tab"
              :class="{ 'sort-tab--active': orderBy === s.value }"
              @click="orderBy = s.value; loadMemories()"
            >
              {{ s.label }}
            </button>
          </div>
        </div>
      </div>
      <!-- 赛道筛选 -->
      <div v-if="genreStore.genres.length > 0" class="genre-filter">
        <button
          class="genre-tag"
          :class="{ 'genre-tag--active': !filterGenreId }"
          @click="filterGenreId = 0; loadMemories()"
        >全部赛道</button>
        <template v-for="genre in genreStore.genres" :key="genre.id">
          <button
            class="genre-tag"
            :class="{ 'genre-tag--active': filterGenreId === genre.id }"
            @click="filterGenreId = genre.id; loadMemories()"
          >{{ genre.icon }} {{ genre.name }}</button>
        </template>
      </div>
    </FadePanel>

    <!-- 记忆列表 -->
    <div v-loading="marketStore.loading" class="market-grid">
      <GlowCard
        v-for="memory in marketStore.memories"
        :key="memory.id"
        hoverable
        class="market-card"
        @click="goDetail(memory.id)"
      >
        <div class="market-card__header">
          <div class="market-card__header-left">
            <el-tag size="small" :type="categoryTagType(memory.category)" effect="dark">
              {{ categoryLabel(memory.category) }}
            </el-tag>
            <span v-if="memory.quality_grade" class="grade-badge" :class="`grade-badge--${memory.quality_grade}`">{{ memory.quality_grade }}</span>
          </div>
          <span class="market-card__price">{{ memory.price }} <small>积分</small></span>
        </div>
        <h3 class="market-card__title">{{ memory.title }}</h3>
        <p class="market-card__desc">{{ memory.description || memory.preview_text?.slice(0, 80) || '暂无描述' }}</p>
        <div class="market-card__stats">
          <div class="stat-item">
            <span class="stat-item__icon">⭐</span>
            <span>{{ (memory.avg_rating || 0).toFixed(1) }}</span>
            <span class="stat-item__count">({{ memory.rating_count || 0 }})</span>
          </div>
          <div class="stat-item">
            <span class="stat-item__icon">📦</span>
            <span>{{ memory.sales_count }} 次购买</span>
          </div>
        </div>
        <div v-if="memory.tags" class="market-card__tags">
          <span v-for="tag in memory.tags.split(',').slice(0, 3)" :key="tag" class="mini-tag">{{ tag.trim() }}</span>
        </div>
      </GlowCard>

      <div v-if="!marketStore.loading && marketStore.memories.length === 0" class="empty-state">
        <div class="empty-state__icon">🏪</div>
        <p>暂无上架记忆</p>
        <span>成为第一个上架记忆的创作者吧</span>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="marketStore.total > 20" class="pagination">
      <el-pagination
        v-model:current-page="page"
        :page-size="20"
        :total="marketStore.total"
        layout="prev, pager, next"
        background
        @current-change="loadMemories"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMarketStore } from '@/store/market'
import { useGenreStore } from '@/store/genre'
import { memoryCategoryOptions } from '@/api/memory'
import GlowCard from '@/components/common/GlowCard.vue'
import FadePanel from '@/components/common/FadePanel.vue'

const router = useRouter()
const marketStore = useMarketStore()
const genreStore = useGenreStore()
const filterCategory = ref('')
const keyword = ref('')
const orderBy = ref('sales')
const page = ref(1)
const filterGenreId = ref(0)

const categoryTabs = [
  { value: '', label: '全部', icon: '📋' },
  ...memoryCategoryOptions.map(o => ({
    value: o.value,
    label: o.label,
    icon: { style: '✍️', character: '👤', worldview: '🌍', plot_preference: '📖' }[o.value] || '📋',
  })),
]

const sortOptions = [
  { value: 'sales', label: '最热门' },
  { value: 'rating', label: '最高分' },
  { value: 'newest', label: '最新' },
]

onMounted(() => {
  loadMemories()
  genreStore.fetchGenreTree()
})

function loadMemories() {
  marketStore.fetchMemories({
    category: filterCategory.value || undefined,
    keyword: keyword.value || undefined,
    order_by: orderBy.value,
    page: page.value,
    page_size: 20,
    genre_id: filterGenreId.value || undefined,
  })
}

function categoryLabel(cat: string) {
  return memoryCategoryOptions.find(o => o.value === cat)?.label || cat
}

function categoryTagType(cat: string): string {
  const map: Record<string, string> = { style: '', character: 'success', worldview: 'warning', plot_preference: 'danger' }
  return map[cat] || ''
}

function goDetail(mid: number) {
  router.push({ name: 'MemoryDetail', params: { mid } })
}
</script>

<style scoped lang="scss">
.memory-market {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.page-desc {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.filter-panel {
  margin-bottom: 24px;
  padding: 16px 20px;
}

.filter-row {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-tabs {
  display: flex;
  gap: 4px;
}

.filter-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.25s ease;

  &:hover {
    background: var(--color-bg-hover);
    color: var(--color-text-primary);
  }

  &--active {
    background: var(--color-primary);
    color: #fff;
    box-shadow: 0 0 12px rgba(124, 140, 248, 0.3);

    &:hover {
      background: var(--color-primary-light);
      color: #fff;
    }
  }

  &__icon { font-size: 14px; }
}

.filter-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background: var(--color-bg-deep);
  border: 1px solid var(--border-glow);
  border-radius: 8px;
  transition: border-color 0.2s;

  &:focus-within {
    border-color: var(--color-primary);
  }

  &__icon { font-size: 13px; }

  &__input {
    border: none;
    background: transparent;
    color: var(--color-text-primary);
    font-size: 13px;
    outline: none;
    width: 160px;

    &::placeholder {
      color: var(--color-text-muted);
    }
  }
}

.sort-tabs {
  display: flex;
  gap: 2px;
  background: var(--color-bg-deep);
  border-radius: 6px;
  padding: 2px;
}

.sort-tab {
  padding: 6px 12px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--color-text-muted);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    color: var(--color-text-secondary);
  }

  &--active {
    background: var(--color-bg-hover);
    color: var(--color-text-primary);
  }
}

.market-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
}

.market-card {
  &__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
  }

  &__price {
    color: var(--color-accent-amber);
    font-weight: 700;
    font-size: 16px;

    small {
      font-size: 11px;
      font-weight: 400;
      opacity: 0.7;
    }
  }

  &__title {
    font-size: 16px;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 6px;
  }

  &__desc {
    font-size: 13px;
    color: var(--color-text-secondary);
    margin-bottom: 12px;
    line-height: 1.5;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  &__stats {
    display: flex;
    gap: 16px;
    margin-bottom: 10px;
  }

  &__tags {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
    padding-top: 10px;
    border-top: 1px solid var(--border-glow);
  }
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--color-text-secondary);

  &__icon { font-size: 13px; }
  &__count { color: var(--color-text-muted); }
}

.mini-tag {
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--color-bg-hover);
  color: var(--color-text-muted);
  font-size: 11px;
}

.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 80px 20px;

  &__icon { font-size: 48px; margin-bottom: 16px; }
  p { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 8px; }
  span { font-size: 13px; color: var(--color-text-muted); }
}

.pagination {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

.market-card__header-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.grade-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 700;
  color: #fff;

  &--S { background: linear-gradient(135deg, #f59e0b, #ef4444); }
  &--A { background: #8b5cf6; }
  &--B { background: #3b82f6; }
  &--C { background: #6b7280; }
  &--D { background: #374151; }
}

.genre-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-glow);
}

.genre-tag {
  padding: 4px 12px;
  border-radius: 16px;
  border: 1px solid var(--border-glow);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  &--active {
    background: var(--color-primary);
    border-color: var(--color-primary);
    color: #fff;
  }
}
</style>
