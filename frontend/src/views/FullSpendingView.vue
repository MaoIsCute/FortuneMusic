<template>
  <div class="page">
    <h1 class="page-title">💰 全握花費（預估）</h1>
    <p class="page-subtitle">依「應募口數 × 單曲單價」回推估算，不是網站上抓得到的實際花費金額</p>

    <template v-if="loaded">
    <ErrorState v-if="loadFailed" />
    <EmptyState v-else-if="isEmpty" />
    <template v-else>
    <!-- 總覽 -->
    <div class="stats-row">
      <el-card class="stat-card">
        <div class="stat-num">¥{{ overall.totalCost.toLocaleString() }}</div>
        <div class="stat-label">總預估花費</div>
      </el-card>
      <el-card class="stat-card">
        <div class="stat-num">{{ overall.totalApplied.toLocaleString() }}</div>
        <div class="stat-label">總應募口數</div>
      </el-card>
      <el-card class="stat-card" v-if="overall.unpricedCount > 0">
        <div class="stat-num warn">{{ overall.unpricedCount }}</div>
        <div class="stat-label">未登記價格的單曲</div>
      </el-card>
    </div>

    <div class="hint-text">💡 標示「未登記價格」的單曲不計入預估花費（避免顯示 ¥0 誤以為免費），可請管理者在後台補登該單曲的單價級距。</div>

    <!-- 依單曲（樹狀） -->
    <el-card class="section">
      <template #header><span>依單曲</span></template>

      <div v-if="tree.length === 0" class="empty">尚無資料</div>

      <el-collapse v-else accordion>
        <el-collapse-item v-for="g in tree" :key="g.group" :name="g.group">
          <template #title>
            <div class="tree-title group-title">
              <span class="tree-name" :style="{ color: GROUP_COLORS[g.group] }">{{ groupLabel(g.group) }}</span>
              <span class="tree-meta">¥{{ g.estimated_cost.toLocaleString() }} &nbsp;/&nbsp; {{ g.applied_count }}口</span>
            </div>
          </template>

          <!-- 單曲層 -->
          <el-collapse accordion class="inner-collapse">
            <el-collapse-item v-for="s in g.singles" :key="s.single_number" :name="s.single_number">
              <template #title>
                <div class="tree-title">
                  <span class="tree-name" :style="{ color: GROUP_COLORS[g.group] }">{{ formatSingle(s.single_name) || `第${s.single_number}單` }}</span>
                  <span class="tree-meta">
                    <span v-if="!s.has_price" class="unpriced-badge">未登記價格</span>
                    <template v-else>¥{{ s.estimated_cost.toLocaleString() }} ({{ s.unit_price.toLocaleString() }}/口) &nbsp;/&nbsp; </template>
                    {{ s.applied_count }}口
                  </span>
                </div>
              </template>

              <!-- 抽次層 -->
              <el-collapse accordion class="inner-collapse">
                <el-collapse-item v-for="r in s.rounds" :key="r.lottery_round" :name="r.lottery_round">
                  <template #title>
                    <div class="tree-title round-title">
                      <span class="tree-name">{{ formatRound(r.lottery_round) }}</span>
                      <span class="tree-meta">
                        <template v-if="s.has_price">¥{{ r.estimated_cost.toLocaleString() }} &nbsp;/&nbsp; </template>
                        {{ r.applied_count }}口
                      </span>
                    </div>
                  </template>

                  <!-- 成員層 -->
                  <div v-for="(m, i) in r.members" :key="`${m.member_name}:${i}`" class="member-row">
                    <span class="member-name" :style="{ color: GROUP_COLORS[g.group], fontWeight: 500 }">{{ m.member_name }}</span>
                    <span class="member-meta">
                      <template v-if="s.has_price">¥{{ m.estimated_cost.toLocaleString() }} &nbsp;/&nbsp; </template>
                      {{ m.applied_count }}口
                    </span>
                  </div>
                </el-collapse-item>
              </el-collapse>
            </el-collapse-item>
          </el-collapse>
        </el-collapse-item>
      </el-collapse>
    </el-card>
    </template>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getFullSpendingTree } from '../api/index'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'

const tree       = ref([])
const loaded     = ref(false)
const loadFailed = ref(false)

const GROUP_LABELS = { nogizaka46: '乃木坂46', sakurazaka46: '櫻坂46', hinatazaka46: '日向坂46' }
const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }
function groupLabel(g) {
  return GROUP_LABELS[g] || g || '—'
}

const isEmpty = computed(() => loaded.value && tree.value.length === 0)

const overall = computed(() => {
  let totalCost = 0, totalApplied = 0, unpricedCount = 0
  for (const g of tree.value) {
    totalCost += g.estimated_cost
    totalApplied += g.applied_count
    for (const s of g.singles) {
      if (!s.has_price) unpricedCount++
    }
  }
  return { totalCost, totalApplied, unpricedCount }
})

function formatSingle(name) {
  return (name ?? '')
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
    .replace(/^アルバム/, '專輯')
}

function formatRound(round) {
  return round ? `${round}抽` : '—'
}

async function load() {
  try {
    const res = await getFullSpendingTree()
    tree.value = res.data ?? []
  } catch {
    loadFailed.value = true
  } finally {
    loaded.value = true
  }
}

onMounted(load)
</script>

<style scoped>
.page-subtitle { color: #888; font-size: 13px; margin: 0 0 20px; }
html.dark .page-subtitle { color: #9aa3b5; }

.stats-row  { display: flex; gap: 16px; margin-bottom: 16px; flex-wrap: wrap; }
.stat-card { flex: 1; min-width: 140px; text-align: center; }
.stat-num  { font-size: 26px; font-weight: bold; color: var(--el-color-primary); }
.stat-num.warn { color: #f97316; }
.stat-label { font-size: 13px; color: #888; margin-top: 4px; }
.hint-text { font-size: 12px; color: #999; margin-bottom: 20px; line-height: 1.5; }
.section { margin-bottom: 24px; }
.empty { color: #999; text-align: center; padding: 32px 0; }

.tree-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding-right: 12px;
}
.tree-name { font-weight: 500; }
.tree-meta { color: #888; font-size: 13px; white-space: nowrap; }
.unpriced-badge {
  display: inline-block;
  background: #fef3c7;
  color: #b45309;
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 11px;
  margin-right: 8px;
}
html.dark .unpriced-badge { background: #78350f; color: #fde68a; }

.group-title .tree-name { font-weight: 700; font-size: 15px; }
.round-title .tree-name { font-weight: normal; color: #555; }

.inner-collapse { margin: 0; }

.member-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 16px;
  font-size: 13px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.member-row:last-child { border-bottom: none; }
.member-name { color: #333; }
.member-meta { color: #888; }
</style>
