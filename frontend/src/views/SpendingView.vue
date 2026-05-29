<template>
  <div class="page">
    <h1 class="page-title">💰 花費統計</h1>

    <!-- 總覽 -->
    <div class="stats-row">
      <el-card class="stat-card">
        <div class="stat-num">¥{{ overall.total_amount?.toLocaleString() ?? '—' }}</div>
        <div class="stat-label">總花費</div>
      </el-card>
      <el-card class="stat-card">
        <div class="stat-num">{{ overall.total_quantity?.toLocaleString() ?? '—' }}</div>
        <div class="stat-label">總張數</div>
      </el-card>
      <el-card class="stat-card">
        <div class="stat-num">{{ overall.purchase_count?.toLocaleString() ?? '—' }}</div>
        <div class="stat-label">筆數</div>
      </el-card>
    </div>

    <!-- 依單曲（樹狀） -->
    <el-card class="section">
      <template #header><span>依單曲</span></template>

      <div v-if="tree.length === 0" class="empty">尚無資料</div>

      <el-collapse v-else accordion>
        <el-collapse-item v-for="s in tree" :key="`${s.single_number}:${s.single_name}`" :name="`${s.single_number}:${s.single_name}`">
          <template #title>
            <div class="tree-title">
              <span class="tree-name">{{ s.single_name }}</span>
              <span class="tree-meta">¥{{ s.total_amount.toLocaleString() }} &nbsp;/&nbsp; {{ s.total_quantity }}張</span>
            </div>
          </template>

          <!-- 抽次層 -->
          <el-collapse accordion class="inner-collapse">
            <el-collapse-item v-for="r in s.rounds" :key="r.lottery_round" :name="r.lottery_round">
              <template #title>
                <div class="tree-title round-title">
                  <span class="tree-name">{{ formatRound(r.lottery_round) }}</span>
                  <span class="tree-meta">¥{{ r.total_amount.toLocaleString() }} &nbsp;/&nbsp; {{ r.total_quantity }}張</span>
                </div>
              </template>

              <!-- 成員層 -->
              <div v-for="m in r.members" :key="m.member_name" class="member-row">
                <span class="member-name">{{ m.member_name }}</span>
                <span class="member-meta">¥{{ m.total_amount.toLocaleString() }} &nbsp;/&nbsp; {{ m.total_quantity }}張</span>
              </div>
            </el-collapse-item>
          </el-collapse>
        </el-collapse-item>
      </el-collapse>
    </el-card>

    <!-- 依成員 -->
    <el-card class="section">
      <template #header><span>依成員</span></template>
      <div v-if="byMember.length === 0" class="empty">尚無資料</div>
      <el-table v-else :data="byMember" stripe>
        <el-table-column prop="member_name" label="成員" />
        <el-table-column prop="total_quantity" label="張數" width="80" align="right" />
        <el-table-column label="金額" width="130" align="right">
          <template #default="{ row }">¥{{ row.total_amount?.toLocaleString() }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getPurchaseOverallStats, getPurchaseTree, getPurchaseStatsByMember } from '../api/index'

const overall  = ref({})
const tree     = ref([])
const byMember = ref([])

function formatRound(round) {
  return round ? `${round}抽` : '—'
}

async function load() {
  const [o, t, mb] = await Promise.all([
    getPurchaseOverallStats(),
    getPurchaseTree(),
    getPurchaseStatsByMember(),
  ])
  overall.value  = o.data
  tree.value     = t.data ?? []
  byMember.value = mb.data ?? []
}

onMounted(load)
</script>

<style scoped>
.stats-row { display: flex; gap: 16px; margin-bottom: 24px; flex-wrap: wrap; }
.stat-card { flex: 1; min-width: 140px; text-align: center; }
.stat-num  { font-size: 26px; font-weight: bold; color: var(--el-color-primary); }
.stat-label { font-size: 13px; color: #888; margin-top: 4px; }
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
