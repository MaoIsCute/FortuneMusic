<template>
  <div class="page">
    <h1 class="page-title">📊 全員統計</h1>

    <template v-if="pageLoaded">
    <el-collapse v-model="quickStartOpen" class="quick-start-collapse">
      <el-collapse-item name="quickStart">
        <template #title><span class="collapse-title">🚀 新手上路：快速上手指南</span></template>
        <div class="quick-start-steps">
          <div class="qs-step">
            <div class="qs-num">1</div>
            <div class="qs-text">
              <div class="qs-title">連結同步工具</div>
              <div class="qs-sub">前往「同步工具」頁面，安裝並連結 Chrome 擴充功能（沒安裝過會有安裝引導）。</div>
              <router-link to="/scrape" class="qs-link">前往同步工具 →</router-link>
            </div>
          </div>
          <div class="qs-step">
            <div class="qs-num">2</div>
            <div class="qs-text">
              <div class="qs-title">開始同步資料</div>
              <div class="qs-sub">點 Chrome 右上角的擴充功能圖示，依畫面指示同步個握／全握紀錄。</div>
            </div>
          </div>
          <div class="qs-step">
            <div class="qs-num">3</div>
            <div class="qs-text">
              <div class="qs-title">查看自己的統計</div>
              <div class="qs-sub">同步完成後，到「個握 ▾ → 個握分析」或「全握 ▾ → 全握分析」查看自己的中選紀錄與分析。</div>
            </div>
          </div>
          <div class="qs-step">
            <div class="qs-num">4</div>
            <div class="qs-text">
              <div class="qs-title">解鎖全員統計</div>
              <div class="qs-sub">貢獻過個握或全握資料後，下方的「個握總表」「全握總表」就會自動解鎖，可以看到所有使用者彙整起來的中選率分析。</div>
            </div>
          </div>
        </div>
        <div class="qs-dismiss" @click="dismissQuickStart">不再顯示</div>
      </el-collapse-item>
    </el-collapse>

    <el-tabs v-model="activeTab">

      <!-- 個握總表 -->
      <el-tab-pane label="個握總表" name="records">
        <LockedState
          v-if="!recordsUnlocked"
          title="貢獻個握資料才能查看"
          sub="個握總表統計了所有使用者的個握中選率，需要你自己也同步過至少一筆個握紀錄才能解鎖。"
        />
        <template v-else>
          <div class="stats-grid">
            <div class="stat-card">
              <div class="stat-label">貢獻人數</div>
              <div class="stat-value">{{ rOverall.contributor_count }}</div>
            </div>
            <div class="stat-card">
              <div class="stat-label">總應募數</div>
              <div class="stat-value">{{ rOverall.total_applied }}</div>
            </div>
            <div class="stat-card">
              <div class="stat-label">總中選數</div>
              <div class="stat-value">{{ rOverall.total_won }}</div>
            </div>
            <div class="stat-card">
              <div class="stat-label">總中選率</div>
              <div class="stat-value highlight">{{ rOverall.win_rate.toFixed(1) }}%</div>
            </div>
          </div>

          <el-collapse v-model="rOpenSections">

            <!-- 各次應募中選率折線圖 -->
            <el-collapse-item v-if="rChartOption.series.length" name="trend">
              <template #title><span class="collapse-title">各次應募中選率比較</span></template>
              <div class="chart-range-btns">
                <button
                  :class="['range-btn', { active: rIsAllSelected }]"
                  @click="rToggleAllLegend"
                >成員全選</button>
                <span class="range-divider">|</span>
                <button
                  v-for="opt in rangeOptions"
                  :key="opt.value"
                  :class="['range-btn', { active: rChartRange === opt.value }]"
                  @click="rChartRange = opt.value"
                >{{ opt.label }}</button>
              </div>
              <div class="chart-filters">
                <el-select v-model="rChartFilterGroup" placeholder="團體（全部）" clearable multiple collapse-tags size="small" style="width:200px" @change="rOnChartGroupChange">
                  <el-option label="乃木坂46" value="nogizaka46">
                    <span :style="{ color: GROUP_COLORS.nogizaka46, fontWeight: 500 }">乃木坂46</span>
                  </el-option>
                  <el-option label="櫻坂46" value="sakurazaka46">
                    <span :style="{ color: GROUP_COLORS.sakurazaka46, fontWeight: 500 }">櫻坂46</span>
                  </el-option>
                  <el-option label="日向坂46" value="hinatazaka46">
                    <span :style="{ color: GROUP_COLORS.hinatazaka46, fontWeight: 500 }">日向坂46</span>
                  </el-option>
                </el-select>
                <el-select v-model="rChartFilterMembers" placeholder="成員（不選 = 全部顯示）" clearable multiple filterable collapse-tags collapse-tags-tooltip size="small" style="width:280px" @change="rApplyChartFilter">
                  <el-option v-for="m in rChartMemberOptions" :key="m.name" :label="m.name" :value="m.name">
                    <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
                  </el-option>
                </el-select>
              </div>
              <v-chart :option="rChartOption" autoresize style="height: 320px;" @legendselectchanged="rOnLegendChange" />
            </el-collapse-item>

            <!-- 各部中選率長條圖 -->
            <el-collapse-item name="session">
              <template #title><span class="collapse-title">各部中選率</span></template>
              <div class="chart-filters">
                <el-select v-model="rBarFilterMember" placeholder="選擇成員" clearable size="small">
                  <el-option v-for="m in rAllMembers" :key="m.name" :label="m.name" :value="m.name">
                    <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
                  </el-option>
                </el-select>
                <el-select v-model="rBarFilterRound" placeholder="選擇抽次" clearable size="small">
                  <el-option v-for="r in rAllRounds" :key="r" :label="formatRound(r)" :value="r" />
                </el-select>
              </div>
              <v-chart v-if="rSessionChartOption.series?.length" :option="rSessionChartOption" autoresize style="height: 300px;" />
              <div v-else class="chart-empty">請選擇篩選條件</div>
            </el-collapse-item>

            <!-- 訂單序號 vs 中選率長條圖 -->
            <el-collapse-item name="sequence">
              <template #title><span class="collapse-title">各筆應募中選率</span></template>
              <div class="chart-filters">
                <el-select v-model="rSeqFilterMember" placeholder="選擇成員" clearable size="small" @change="rFetchSeqChart">
                  <el-option v-for="m in rAllMembers" :key="m.name" :label="m.name" :value="m.name">
                    <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
                  </el-option>
                </el-select>
                <el-select v-model="rSeqFilterSession" placeholder="選擇部數" clearable size="small" @change="rFetchSeqChart">
                  <el-option v-for="s in rAllSessions" :key="s" :label="s" :value="s" />
                </el-select>
                <el-select v-model="rSeqFilterRound" placeholder="選擇抽次" clearable size="small" @change="rFetchSeqChart">
                  <el-option v-for="r in rAllRounds" :key="r" :label="formatRound(r)" :value="r" />
                </el-select>
              </div>
              <v-chart v-if="rSeqChartOption.series?.length" :option="rSeqChartOption" autoresize style="height: 300px;" />
              <div v-else class="chart-empty">請選擇篩選條件</div>
            </el-collapse-item>

            <!-- 排行榜 -->
            <el-collapse-item name="ranking">
              <template #title><span class="collapse-title">排行榜</span></template>
              <div class="chart-filters">
                <el-select v-model="rRankFilterGroup" placeholder="團體（全部）" clearable size="small" @change="rLoadRankings">
                  <el-option label="乃木坂46" value="nogizaka46">
                    <span :style="{ color: GROUP_COLORS.nogizaka46, fontWeight: 500 }">乃木坂46</span>
                  </el-option>
                  <el-option label="櫻坂46" value="sakurazaka46">
                    <span :style="{ color: GROUP_COLORS.sakurazaka46, fontWeight: 500 }">櫻坂46</span>
                  </el-option>
                  <el-option label="日向坂46" value="hinatazaka46">
                    <span :style="{ color: GROUP_COLORS.hinatazaka46, fontWeight: 500 }">日向坂46</span>
                  </el-option>
                </el-select>
              </div>

              <p class="rank-sub-title">依成員排行榜</p>
              <el-table :data="rByMember" stripe max-height="360">
                <el-table-column label="團體" min-width="70">
                  <template #default="{ row }"><span :style="{ color: GROUP_COLORS[row.group] }">{{ groupLabel(row.group) }}</span></template>
                </el-table-column>
                <el-table-column label="成員" min-width="100" sortable sort-by="member_name">
                  <template #default="{ row }"><span :style="{ color: GROUP_COLORS[row.group], fontWeight: 500 }">{{ row.member_name }}</span></template>
                </el-table-column>
                <el-table-column prop="total_applied" label="應募" width="90" sortable />
                <el-table-column prop="total_won" label="中選" width="90" sortable />
                <el-table-column label="中選率" width="100" sortable :sort-by="row => row.win_rate">
                  <template #default="{ row }"><span :class="rateClass(row.win_rate)">{{ row.win_rate }}%</span></template>
                </el-table-column>
              </el-table>

              <p class="rank-sub-title">依單曲排行榜</p>
              <el-table :data="rBySingle" stripe max-height="360">
                <el-table-column label="團體" min-width="70">
                  <template #default="{ row }"><span :style="{ color: GROUP_COLORS[row.group] }">{{ groupLabel(row.group) }}</span></template>
                </el-table-column>
                <el-table-column label="單曲" min-width="220">
                  <template #default="{ row }"><span :style="{ color: GROUP_COLORS[row.group] }">{{ formatSingle(row.single_name) }}</span></template>
                </el-table-column>
                <el-table-column prop="total_applied" label="應募" width="90" sortable />
                <el-table-column prop="total_won" label="中選" width="90" sortable />
                <el-table-column label="中選率" width="100" sortable :sort-by="row => row.win_rate">
                  <template #default="{ row }"><span :class="rateClass(row.win_rate)">{{ row.win_rate }}%</span></template>
                </el-table-column>
              </el-table>
            </el-collapse-item>

            <!-- 成員手風琴列表 -->
            <el-collapse-item name="members">
              <template #title><span class="collapse-title">成員列表</span></template>
              <div class="member-list-header">
                <el-select
                  v-model="rFilterMembers"
                  multiple
                  clearable
                  collapse-tags
                  collapse-tags-tooltip
                  placeholder="顯示特定成員（不選 = 全部）"
                  size="small"
                  class="member-filter-select"
                >
                  <el-option v-for="m in rAllMembers" :key="m.name" :label="m.name" :value="m.name">
                    <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
                  </el-option>
                </el-select>
                <button
                  :class="['range-btn', { active: rShowActiveOnly }]"
                  @click="rShowActiveOnly = !rShowActiveOnly"
                >在籍成員</button>
              </div>

              <div class="member-list">
                <div
                  v-for="[group, data] in rGroupedMembers"
                  :key="group"
                  class="group-card"
                >
                  <div class="group-header" @click="toggleGroupSection(group)">
                    <span class="group-name" :style="{ color: GROUP_COLORS[group] }">{{ groupLabel(group) }}</span>
                    <span class="member-summary">
                      {{ data.totalApplied }} 應 / {{ data.totalWon }} 中
                      <span class="rate">{{ calcRate(data.totalWon, data.totalApplied) }}%</span>
                    </span>
                    <span class="chevron">{{ isGroupExpanded(group) ? '▲' : '▼' }}</span>
                  </div>

                  <div v-if="isGroupExpanded(group)" class="group-body">
                    <div
                      v-for="[memberName, member] in data.members"
                      :key="memberName"
                      class="member-card"
                    >
                      <div class="member-header" @click="toggleMember(memberName)">
                        <span class="member-name">{{ memberName }}</span>
                        <span class="member-summary">
                          {{ member.totalApplied }} 應 / {{ member.totalWon }} 中
                          <span class="rate">{{ calcRate(member.totalWon, member.totalApplied) }}%</span>
                        </span>
                        <span class="chevron">{{ rExpandedMembers[memberName] ? '▲' : '▼' }}</span>
                      </div>

                      <div v-if="rExpandedMembers[memberName]" class="member-body">
                        <div
                          v-for="[singleNum, single] in sortedSingles(member.singles)"
                          :key="singleNum"
                          class="single-card"
                        >
                          <div class="single-header" @click="toggleSingle(memberName, singleNum)">
                            <span class="single-name">{{ formatSingle(single.singleName) }}</span>
                            <span class="single-summary">
                              {{ single.totalApplied }} 應 / {{ single.totalWon }} 中
                              <span class="rate">{{ calcRate(single.totalWon, single.totalApplied) }}%</span>
                            </span>
                            <span class="chevron">{{ isSingleExpanded(memberName, singleNum) ? '▲' : '▼' }}</span>
                          </div>

                          <div v-if="isSingleExpanded(memberName, singleNum)" class="single-body">
                            <div
                              v-for="[round, roundData] in sortedRounds(single.rounds)"
                              :key="round"
                              class="round-card"
                            >
                              <div class="round-header" @click="toggleRound(memberName, singleNum, round)">
                                <span class="round-label">{{ formatRound(round) }}</span>
                                <span class="round-summary">
                                  {{ roundData.totalApplied }} 應 / {{ roundData.totalWon }} 中
                                  <span class="rate">{{ calcRate(roundData.totalWon, roundData.totalApplied) }}%</span>
                                </span>
                                <span class="chevron">{{ isRoundExpanded(memberName, singleNum, round) ? '▲' : '▼' }}</span>
                              </div>
                              <table v-if="isRoundExpanded(memberName, singleNum, round)" class="detail-table">
                                <thead>
                                  <tr>
                                    <th>日期</th>
                                    <th>部數</th>
                                    <th>應募</th>
                                    <th>中選</th>
                                    <th>中選率</th>
                                  </tr>
                                </thead>
                                <tbody>
                                  <tr
                                    v-for="row in sortedRows(roundData.rows)"
                                    :key="row.event_date + row.session"
                                  >
                                    <td>{{ row.event_date }}</td>
                                    <td>{{ row.session }}</td>
                                    <td>{{ row.total_applied }}</td>
                                    <td>{{ row.total_won }}</td>
                                    <td>
                                      <span :class="rateClass(row.win_rate)">
                                        {{ row.win_rate.toFixed(1) }}%
                                      </span>
                                    </td>
                                  </tr>
                                </tbody>
                              </table>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </el-collapse-item>

          </el-collapse>
        </template>
      </el-tab-pane>

      <!-- 全握總表 -->
      <el-tab-pane label="全握總表" name="full">
        <LockedState
          v-if="!fullUnlocked"
          title="貢獻全握資料才能查看"
          sub="全握總表統計了所有使用者的全握中選率，需要你自己也同步過至少一筆全握紀錄才能解鎖。"
        />
        <template v-else>
          <div class="stats-grid">
            <div class="stat-card">
              <div class="stat-label">貢獻人數</div>
              <div class="stat-value">{{ fContributorCount }}</div>
            </div>
            <div class="stat-card">
              <div class="stat-label">總應募數</div>
              <div class="stat-value">{{ fOverall.total_applied }}</div>
            </div>
            <div class="stat-card">
              <div class="stat-label">總中選數</div>
              <div class="stat-value">{{ fOverall.total_won }}</div>
            </div>
            <div class="stat-card">
              <div class="stat-label">總中選率</div>
              <div class="stat-value highlight">{{ fOverallRate }}%</div>
            </div>
          </div>

          <el-collapse v-model="fOpenSections">

            <!-- 類型分析 -->
            <el-collapse-item v-if="fByType.length" name="type">
              <template #title><span class="collapse-title">類型分析</span></template>
              <el-table table-layout="auto" :data="fByType" stripe>
                <el-table-column prop="event_type" label="類型" min-width="60" sortable />
                <el-table-column label="場地" min-width="200" sortable :sort-by="row => row.venue || ''">
                  <template #default="{ row }">{{ row.venue || '—' }}</template>
                </el-table-column>
                <el-table-column prop="total_applied" label="應募" min-width="70" sortable />
                <el-table-column prop="total_won" label="中選" min-width="70" sortable />
                <el-table-column prop="win_rate_num" label="中選率" min-width="80" sortable>
                  <template #default="{ row }">
                    <span :class="rateClass(row.win_rate)">{{ row.win_rate }}%</span>
                  </template>
                </el-table-column>
              </el-table>
            </el-collapse-item>

            <!-- 地區分析 -->
            <el-collapse-item name="region">
              <template #title><span class="collapse-title">地區分析（關東場 vs 地方場）</span></template>
              <div class="filters">
                <el-select v-model="fRegionFilterGroup" placeholder="團體" clearable style="width:120px" @change="fLoadRegionStats">
                  <el-option label="乃木坂46" value="nogizaka46">
                    <span :style="{ color: GROUP_COLORS.nogizaka46, fontWeight: 500 }">乃木坂46</span>
                  </el-option>
                  <el-option label="櫻坂46" value="sakurazaka46">
                    <span :style="{ color: GROUP_COLORS.sakurazaka46, fontWeight: 500 }">櫻坂46</span>
                  </el-option>
                  <el-option label="日向坂46" value="hinatazaka46">
                    <span :style="{ color: GROUP_COLORS.hinatazaka46, fontWeight: 500 }">日向坂46</span>
                  </el-option>
                </el-select>
              </div>
              <el-table :data="fRegionStats" stripe>
                <el-table-column label="地區" min-width="90">
                  <template #default="{ row }">
                    <span :style="{ color: REGION_COLORS[row.region] || '#666', fontWeight: 500 }">{{ row.region }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="total_applied" label="應募" min-width="80" sortable />
                <el-table-column prop="total_won" label="中選" min-width="80" sortable />
                <el-table-column prop="win_rate_num" label="中選率" min-width="90" sortable>
                  <template #default="{ row }">
                    <span :class="rateClass(row.win_rate)">{{ row.win_rate }}%</span>
                  </template>
                </el-table-column>
              </el-table>
              <div class="hint-text">僅統計「実体」場次；地區依場地名稱判斷（關東：幕張メッセ・パシフィコ横浜等／地方：京都パルスプラザ・なごや等），新場地上線後才會被歸類，若顯示「その他」代表尚未登記到判斷清單。</div>
            </el-collapse-item>

            <!-- 成員統計 -->
            <el-collapse-item name="member">
              <template #title><span class="collapse-title">成員統計</span></template>
              <div class="filters">
                <el-select v-model="fMemberFilterGroup" placeholder="團體" clearable style="width:120px" @change="fOnMemberGroupChange">
                  <el-option label="乃木坂46" value="nogizaka46">
                    <span :style="{ color: GROUP_COLORS.nogizaka46, fontWeight: 500 }">乃木坂46</span>
                  </el-option>
                  <el-option label="櫻坂46" value="sakurazaka46">
                    <span :style="{ color: GROUP_COLORS.sakurazaka46, fontWeight: 500 }">櫻坂46</span>
                  </el-option>
                  <el-option label="日向坂46" value="hinatazaka46">
                    <span :style="{ color: GROUP_COLORS.hinatazaka46, fontWeight: 500 }">日向坂46</span>
                  </el-option>
                </el-select>
                <el-select v-model="fMemberFilterType" placeholder="類型" clearable style="width:100px" @change="fOnMemberTypeChange">
                  <el-option label="実体" value="実体" />
                  <el-option label="線上" value="線上" />
                </el-select>
                <el-select v-model="fMemberFilterRegion" placeholder="地區（全部）" clearable style="width:120px"
                  :disabled="fMemberFilterType === '線上'" @change="fOnMemberRegionChange">
                  <el-option label="関東" value="関東">
                    <span :style="{ color: REGION_COLORS['関東'], fontWeight: 500 }">関東</span>
                  </el-option>
                  <el-option label="地方" value="地方">
                    <span :style="{ color: REGION_COLORS['地方'], fontWeight: 500 }">地方</span>
                  </el-option>
                </el-select>
                <el-select v-model="fMemberFilterVenue" placeholder="場地" clearable style="width:160px"
                  :disabled="fMemberFilterType === '線上'" @change="fLoadMemberStats">
                  <el-option v-for="v in fMemberVenueOptions" :key="v" :label="v" :value="v" />
                </el-select>
                <el-checkbox-group v-model="fMemberRowModes">
                  <el-checkbox value="single">單人列</el-checkbox>
                  <el-checkbox value="multi">多人列</el-checkbox>
                </el-checkbox-group>
              </div>
              <el-table :data="fFilteredMemberStats" stripe max-height="400">
                <el-table-column label="成員" sortable sort-by="member_name">
                  <template #default="{ row }">
                    <span :style="{ color: GROUP_COLORS[row.group], fontWeight: 500 }">{{ row.member_name }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="total_applied" label="應募" width="80" sortable />
                <el-table-column prop="total_won" label="中選" width="80" sortable />
                <el-table-column prop="win_rate_num" label="中選率" width="90" sortable>
                  <template #default="{ row }">
                    <span :class="rateClass(row.win_rate)">{{ row.win_rate }}%</span>
                  </template>
                </el-table-column>
              </el-table>
            </el-collapse-item>

            <!-- 成員詳細分析 -->
            <el-collapse-item name="detail">
              <template #title><span class="collapse-title">成員詳細分析</span></template>

              <div class="detail-filters">
                <el-select v-model="fDetailFilterGroup" placeholder="團體" clearable style="width:120px" @change="fOnDetailGroupChange">
                  <el-option label="乃木坂46" value="nogizaka46">
                    <span :style="{ color: GROUP_COLORS.nogizaka46, fontWeight: 500 }">乃木坂46</span>
                  </el-option>
                  <el-option label="櫻坂46" value="sakurazaka46">
                    <span :style="{ color: GROUP_COLORS.sakurazaka46, fontWeight: 500 }">櫻坂46</span>
                  </el-option>
                  <el-option label="日向坂46" value="hinatazaka46">
                    <span :style="{ color: GROUP_COLORS.hinatazaka46, fontWeight: 500 }">日向坂46</span>
                  </el-option>
                </el-select>
                <el-select v-model="fDetailMember" placeholder="選擇成員" clearable style="width:160px" @change="fLoadDetail">
                  <el-option v-for="m in fDetailMemberOptions" :key="m.name" :label="m.name" :value="m.name">
                  <span :style="{ color: GROUP_COLORS[m.group] }">{{ m.name }}</span>
                </el-option>
                </el-select>
                <el-checkbox v-model="fDetailActiveOnly" @change="fOnDetailActiveOnlyChange">只顯示現役成員</el-checkbox>
                <el-select v-model="fDetailType" style="width:120px" @change="fOnDetailTypeChange">
                  <el-option label="実体" value="実体" />
                  <el-option label="線上" value="線上" />
                </el-select>
                <el-select v-model="fDetailRegion" placeholder="地區（全部）" clearable style="width:120px"
                  :disabled="fDetailType === '線上'" @change="fOnDetailRegionChange">
                  <el-option label="関東" value="関東">
                    <span :style="{ color: REGION_COLORS['関東'], fontWeight: 500 }">関東</span>
                  </el-option>
                  <el-option label="地方" value="地方">
                    <span :style="{ color: REGION_COLORS['地方'], fontWeight: 500 }">地方</span>
                  </el-option>
                </el-select>
                <el-select v-model="fDetailVenue" :placeholder="fDetailType === '線上' ? '無' : '場地（全部）'" clearable style="width:140px"
                  :disabled="fDetailType === '線上'" @change="fLoadDetail">
                  <el-option v-for="v in fDetailVenueOptions" :key="v" :label="v" :value="v" />
                </el-select>
                <el-checkbox-group v-model="fSelectedRounds">
                  <el-checkbox :value="1">1抽</el-checkbox>
                  <el-checkbox :value="1.5">1.5抽</el-checkbox>
                  <el-checkbox :value="2">2抽</el-checkbox>
                </el-checkbox-group>
              </div>

              <div v-if="!fDetailMember" class="empty">請先選擇成員</div>
              <div v-else-if="fDetailLoading" class="empty">載入中...</div>
              <div v-else-if="fDetailRows.length === 0" class="empty">無資料</div>
              <el-table v-else :data="fDetailRows" stripe border table-layout="auto">
                <el-table-column label="單曲" width="90" fixed>
                  <template #default="{ row }">{{ formatSingle(row.single_name) || row.single_number + '單' }}</template>
                </el-table-column>
                <el-table-column label="場地" min-width="120">
                  <template #default="{ row }">{{ row.venue || '—' }}</template>
                </el-table-column>
                <el-table-column label="搭檔" width="130">
                  <template #default="{ row }">
                    <span v-if="row.partner" class="partner-name">{{ row.partner }}</span>
                    <span v-else class="text-muted">—</span>
                  </template>
                </el-table-column>
                <el-table-column
                  v-for="session in fDetailSessions"
                  :key="session"
                  :label="session || '—'"
                  align="center"
                >
                  <el-table-column
                    v-for="round in fSelectedRoundsSorted"
                    :key="round"
                    :label="round + '抽'"
                    align="center"
                    width="80"
                  >
                    <template #default="{ row }">
                      <template v-if="row.cells[`${session}:${round}`]">
                        <span :class="rateClass((row.cells[`${session}:${round}`].won / row.cells[`${session}:${round}`].applied * 100).toFixed(1))">
                          {{ (row.cells[`${session}:${round}`].won / row.cells[`${session}:${round}`].applied * 100).toFixed(1) }}%
                        </span>
                        <div class="detail-sub">{{ row.cells[`${session}:${round}`].won }}/{{ row.cells[`${session}:${round}`].applied }}</div>
                      </template>
                      <span v-else class="text-muted">—</span>
                    </template>
                  </el-table-column>
                </el-table-column>
              </el-table>
            </el-collapse-item>

          </el-collapse>
        </template>
      </el-tab-pane>

    </el-tabs>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useDataStore } from '../stores/data'
import { useThemeStore } from '../stores/theme'
import LockedState from '../components/LockedState.vue'
import { getMemberInfo, sortMembersByGroupAndGen, memberOrderIndex } from '../utils/members'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import {
  getGlobalOverallStats, getGlobalDetailStats, getGlobalOrderSequenceStats,
  getGlobalStatsByMember, getGlobalStatsBySingle,
  getFullOverallStats,
  getGlobalFullOverallStats, getGlobalFullStatsByMember, getGlobalFullStatsByRegion, getGlobalFullDetailStats,
} from '../api/index'

use([LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const GROUP_LABELS = { nogizaka46: '乃木坂46', sakurazaka46: '櫻坂46', hinatazaka46: '日向坂46' }
const GROUP_COLORS = { nogizaka46: '#9333ea', sakurazaka46: '#ec4899', hinatazaka46: '#0ea5e9' }
const REGION_COLORS = { '関東': '#2563eb', '地方': '#f97316', 'その他': '#9ca3af' }
// 跟後端 handlers/full_stats.go 的 kantoVenues/regionalVenues 對照表一致，僅供前端場地下拉篩選用
const REGION_VENUES = {
  '関東': ['幕張メッセ', 'パシフィコ横浜', '東京', '東京ビッグサイト'],
  '地方': ['京都パルスプラザ', 'ポートメッセなごや', '地方', 'インテックス大阪'],
}

function groupLabel(g) {
  return GROUP_LABELS[g] || g || '—'
}
function rateClass(rate) {
  if (rate >= 80) return 'rate high'
  if (rate >= 40) return 'rate mid'
  return 'rate low'
}
function formatSingle(singleName) {
  if (!singleName) return ''
  return singleName
    .replace(/(\d+)(?:st|nd|rd|th)シングル/, (_, n) => `${n}單`)
    .replace(/(\d+)(?:st|nd|rd|th)アルバム/, (_, n) => `${n}專`)
    .replace(/^アルバム/, '專輯')
}

const dataStore  = useDataStore()
const themeStore = useThemeStore()
const ct = computed(() => themeStore.isDark
  ? { text: '#d4d8e3', sub: '#9aa3b5', line: '#3a3f5c' }
  : { text: '#555',    sub: '#888',    line: '#e8e8e8' }
)

const activeTab        = ref('records')
const pageLoaded       = ref(false)
const recordsUnlocked  = ref(false)
const fullUnlocked     = ref(false)

const QS_DISMISS_KEY = 'dashboardQuickStartDismissed'
const quickStartOpen = ref(localStorage.getItem(QS_DISMISS_KEY) ? [] : ['quickStart'])
function dismissQuickStart() {
  quickStartOpen.value = []
  localStorage.setItem(QS_DISMISS_KEY, '1')
}

onMounted(async () => {
  recordsUnlocked.value = dataStore.hasData === true
  try {
    const fr = await getFullOverallStats()
    fullUnlocked.value = (fr.data?.overall?.total_applied ?? 0) > 0
  } catch {
    fullUnlocked.value = false
  }

  const tasks = []
  if (recordsUnlocked.value) tasks.push(rLoadStats())
  if (fullUnlocked.value) tasks.push(fLoadStats())
  await Promise.all(tasks)

  pageLoaded.value = true
})

// ══════════════════════════════════════════════════════════════
// 個握總表
// ══════════════════════════════════════════════════════════════

const rOpenSections = ref(['trend', 'session', 'sequence', 'ranking', 'members'])
const rOverall = ref({ total_applied: 0, total_won: 0, win_rate: 0, contributor_count: 0 })
const rRows    = ref([])
const rByMember = ref([])
const rBySingle = ref([])
const rRankFilterGroup = ref('')
const rExpandedMembers = ref({})
const rExpandedSingles = ref({})
const rExpandedRounds  = ref({})

async function rLoadStats() {
  const [ov, detail] = await Promise.all([getGlobalOverallStats(), getGlobalDetailStats()])
  rOverall.value = ov.data
  rRows.value    = detail.data ?? []
  await rLoadRankings()
}

async function rLoadRankings() {
  const params = {}
  if (rRankFilterGroup.value) params.group = rRankFilterGroup.value
  const [m, s] = await Promise.all([getGlobalStatsByMember(params), getGlobalStatsBySingle(params)])
  rByMember.value = m.data ?? []
  rBySingle.value = s.data ?? []
}

// flat rows → member → singleKey → round → rows
const rMemberMap = computed(() => {
  const map = {}
  for (const row of rRows.value) {
    if (!map[row.member_name]) {
      map[row.member_name] = {
        singles: {}, totalApplied: 0, totalWon: 0,
        group: getMemberInfo(row.member_name).group || row.group || '',
      }
    }
    const m = map[row.member_name]
    m.totalApplied += row.total_applied
    m.totalWon     += row.total_won

    const singleKey = row.single_number > 0
      ? String(row.single_number)
      : `album::${row.single_name}`

    if (!m.singles[singleKey]) {
      m.singles[singleKey] = {
        singleName:   row.single_name,
        singleNumber: row.single_number,
        minEventDate: row.event_date,
        rounds:       {},
        totalApplied: 0,
        totalWon:     0,
      }
    } else {
      m.singles[singleKey].singleName = row.single_name
      if (row.event_date < m.singles[singleKey].minEventDate) {
        m.singles[singleKey].minEventDate = row.event_date
      }
    }
    const s = m.singles[singleKey]
    s.totalApplied += row.total_applied
    s.totalWon     += row.total_won

    const roundKey = row.lottery_round || '—'
    if (!s.rounds[roundKey]) s.rounds[roundKey] = { rows: [], totalApplied: 0, totalWon: 0 }
    s.rounds[roundKey].rows.push(row)
    s.rounds[roundKey].totalApplied += row.total_applied
    s.rounds[roundKey].totalWon     += row.total_won
  }
  return map
})

const rShowActiveOnly = ref(false)
const rFilterMembers  = ref([])

const rSortedMembers = computed(() =>
  Object.entries(rMemberMap.value)
    .filter(([name]) => {
      if (rShowActiveOnly.value && !(getMemberInfo(name).active ?? true)) return false
      if (rFilterMembers.value.length && !rFilterMembers.value.includes(name)) return false
      return true
    })
    .sort(([a], [b]) => {
      const ga = getMemberInfo(a).gen ?? 99
      const gb = getMemberInfo(b).gen ?? 99
      if (ga !== gb) return ga - gb
      return memberOrderIndex(a) - memberOrderIndex(b)
    })
)

// 團體 → （團體內沿用 rSortedMembers 已經排好的期別→五十音順序）
const RECORD_GROUP_ORDER = { nogizaka46: 0, sakurazaka46: 1, hinatazaka46: 2 }
const rGroupedMembers = computed(() => {
  const buckets = {}
  for (const entry of rSortedMembers.value) {
    const g = entry[1].group || ''
    if (!buckets[g]) buckets[g] = { totalApplied: 0, totalWon: 0, members: [] }
    buckets[g].members.push(entry)
    buckets[g].totalApplied += entry[1].totalApplied
    buckets[g].totalWon     += entry[1].totalWon
  }
  return Object.entries(buckets).sort(([a], [b]) => (RECORD_GROUP_ORDER[a] ?? 9) - (RECORD_GROUP_ORDER[b] ?? 9))
})

const rExpandedGroups = ref({})
function isGroupExpanded(group) {
  return rExpandedGroups.value[group] !== false
}
function toggleGroupSection(group) {
  rExpandedGroups.value[group] = !isGroupExpanded(group)
}

function sortedSingles(singles) {
  return Object.entries(singles).sort(([, a], [, b]) =>
    parseDate(b.minEventDate) - parseDate(a.minEventDate)
  )
}

function sortedRounds(rounds) {
  return Object.entries(rounds).sort(([a], [b]) => {
    const na = parseInt(a.match(/\d+/)?.[0] ?? 0)
    const nb = parseInt(b.match(/\d+/)?.[0] ?? 0)
    return na - nb
  })
}

function sortedRows(rowList) {
  return [...rowList].sort((a, b) => {
    const da = parseDate(a.event_date)
    const db = parseDate(b.event_date)
    if (da - db !== 0) return da - db
    return a.session.localeCompare(b.session, 'ja')
  })
}

function parseDate(str) {
  const p = str.split('/')
  if (p.length === 3) return new Date(p[0], p[1] - 1, p[2])
  if (p.length === 2) return new Date(2000, p[0] - 1, p[1])
  return new Date(0)
}

function toggleMember(name) {
  rExpandedMembers.value[name] = !rExpandedMembers.value[name]
}
function toggleSingle(memberName, singleName) {
  const key = `${memberName}::${singleName}`
  rExpandedSingles.value[key] = !rExpandedSingles.value[key]
}
function isSingleExpanded(memberName, singleName) {
  return !!rExpandedSingles.value[`${memberName}::${singleName}`]
}
function toggleRound(memberName, singleName, round) {
  const key = `${memberName}::${singleName}::${round}`
  rExpandedRounds.value[key] = !rExpandedRounds.value[key]
}
function isRoundExpanded(memberName, singleName, round) {
  return !!rExpandedRounds.value[`${memberName}::${singleName}::${round}`]
}

function formatRound(round) {
  return round ? `${round}抽` : ''
}
function calcRate(won, applied) {
  if (!applied) return '0.0'
  return (won / applied * 100).toFixed(1)
}

// ── 訂單序號圖篩選 ───────────────────────────────────────
const rSeqFilterMember  = ref('')
const rSeqFilterSession = ref('')
const rSeqFilterRound   = ref('')
const rSeqData          = ref([])

const rAllSessions = computed(() => {
  const set = new Set()
  for (const row of rRows.value) set.add(row.session)
  return [...set].sort((a, b) =>
    parseInt(a.match(/\d+/)?.[0] ?? 0) - parseInt(b.match(/\d+/)?.[0] ?? 0)
  )
})

async function rFetchSeqChart() {
  if (!rSeqFilterMember.value && !rSeqFilterSession.value && !rSeqFilterRound.value) {
    rSeqData.value = []
    return
  }
  const params = {}
  if (rSeqFilterMember.value)  params.member  = rSeqFilterMember.value
  if (rSeqFilterSession.value) params.session = rSeqFilterSession.value
  if (rSeqFilterRound.value)   params.round   = rSeqFilterRound.value
  const res = await getGlobalOrderSequenceStats(params)
  rSeqData.value = res.data ?? []
}

const rSeqChartOption = computed(() => {
  if (!rSeqData.value.length) return {}
  const labels = rSeqData.value.map(d => d.position)
  const data   = rSeqData.value.map(d => ({
    value:   d.win_rate,
    applied: d.applied,
    won:     d.won,
  }))
  const c = ct.value
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      formatter(params) {
        const d = data[params[0].dataIndex]
        return `${params[0].name}<br/>中選率：${params[0].value}%<br/>應募：${d.applied}　中選：${d.won}`
      },
    },
    grid: { top: 16, right: 24, bottom: 40, left: 54 },
    xAxis: { type: 'category', data: labels, axisLabel: { color: c.text }, axisLine: { lineStyle: { color: c.line } } },
    yAxis: {
      type: 'value', min: 0, max: 100,
      axisLabel: { formatter: '{value}%', color: c.text },
      splitLine: { lineStyle: { type: 'dashed', color: c.line } },
    },
    series: [{
      type: 'bar',
      data: data.map(d => ({
        value: d.value,
        itemStyle: { color: d.value >= 80 ? '#52c41a' : d.value >= 40 ? '#faad14' : '#ff4d4f' },
      })),
      label: { show: true, position: 'top', formatter: '{c}%', fontSize: 12, color: c.text },
    }],
  }
})

// ── 各部長條圖篩選 ───────────────────────────────────────
const rBarFilterMember = ref('')
const rBarFilterRound  = ref('')

const rAllMembers = computed(() => {
  const nameGroupMap = new Map()
  rRows.value.forEach(r => nameGroupMap.set(r.member_name, r.group || ''))
  return sortMembersByGroupAndGen([...nameGroupMap.entries()].map(([name, group]) => ({ name, group })))
})

const rAllRounds = computed(() => {
  const set = new Set()
  for (const row of rRows.value) {
    if (row.lottery_round) set.add(row.lottery_round)
  }
  return [...set].sort((a, b) => a - b)
})

const rSessionChartOption = computed(() => {
  const filtered = rRows.value.filter(row => {
    if (rBarFilterMember.value && row.member_name !== rBarFilterMember.value) return false
    if (rBarFilterRound.value && row.lottery_round !== rBarFilterRound.value) return false
    return true
  })

  if (filtered.length === 0) return {}

  const agg = {}
  for (const row of filtered) {
    if (!agg[row.session]) agg[row.session] = { applied: 0, won: 0 }
    agg[row.session].applied += row.total_applied
    agg[row.session].won     += row.total_won
  }

  const sessions = Object.keys(agg).sort((a, b) =>
    parseInt(a.match(/\d+/)?.[0] ?? 0) - parseInt(b.match(/\d+/)?.[0] ?? 0)
  )

  const data = sessions.map(s => {
    const d = agg[s]
    const rate = d.applied ? parseFloat((d.won / d.applied * 100).toFixed(1)) : 0
    return { value: rate, applied: d.applied, won: d.won }
  })

  const c = ct.value
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      formatter(params) {
        const d = data[params[0].dataIndex]
        return `${params[0].name}<br/>中選率：${params[0].value}%<br/>應募：${d.applied}　中選：${d.won}`
      },
    },
    grid: { top: 16, right: 24, bottom: 40, left: 54 },
    xAxis: { type: 'category', data: sessions, axisLabel: { color: c.text }, axisLine: { lineStyle: { color: c.line } } },
    yAxis: {
      type: 'value', min: 0, max: 100,
      axisLabel: { formatter: '{value}%', color: c.text },
      splitLine: { lineStyle: { type: 'dashed', color: c.line } },
    },
    series: [{
      type: 'bar',
      data: data.map(d => ({
        value: d.value,
        itemStyle: { color: d.value >= 80 ? '#52c41a' : d.value >= 40 ? '#faad14' : '#ff4d4f' },
      })),
      label: { show: true, position: 'top', formatter: '{c}%', fontSize: 12, color: c.text },
    }],
  }
})

// ── 折線圖 ──────────────────────────────────────────────
const CHART_COLORS = ['#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de', '#3ba272', '#fc8452', '#9a60b4', '#ea7ccc']

const rangeOptions = [
  { label: '前3抽', value: 3 },
  { label: '前6抽', value: 6 },
  { label: '全部',  value: 0 },
]
const rChartRange     = ref(0)
const rLegendSelected = ref({})

// 團體/成員下拉：成員數一多，直接在圖表 legend 裡點選很難快速找到人，
// 改用團體先縮小範圍、成員下拉多選（可搜尋）來決定折線圖要顯示誰
const rChartFilterGroup   = ref([])
const rChartFilterMembers = ref([])

const rChartMemberOptions = computed(() =>
  rChartFilterGroup.value.length
    ? rAllMembers.value.filter(m => rChartFilterGroup.value.includes(m.group))
    : rAllMembers.value
)

function rOnChartGroupChange() {
  const allowed = new Set(rChartMemberOptions.value.map(m => m.name))
  rChartFilterMembers.value = rChartFilterMembers.value.filter(n => allowed.has(n))
  rApplyChartFilter()
}

// 決定目前 legend 該顯示誰：成員下拉有選就用成員選的那幾個（最精確）；
// 沒選成員但有選團體，退回用「該團體全部成員」當範圍（rChartMemberOptions 已經是團體篩過的選項）；
// 兩個都沒選才是真的全部顯示。原本只看 rChartFilterMembers，只選團體不選成員時等於沒篩到，
// 圖表還是畫出所有團體的成員（見 RecordsAnalysisView.vue 同一套邏輯）
function rChartAllowedNames() {
  if (rChartFilterMembers.value.length > 0) return new Set(rChartFilterMembers.value)
  if (rChartFilterGroup.value.length > 0) return new Set(rChartMemberOptions.value.map(m => m.name))
  return null
}

function rApplyChartFilter() {
  const allowed = rChartAllowedNames()
  const sel = {}
  for (const name of Object.keys(rLegendSelected.value)) {
    sel[name] = name === '全部' || !allowed || allowed.has(name)
  }
  rLegendSelected.value = sel
}

watch(rMemberMap, (map) => {
  const allowed = rChartAllowedNames()
  const sel = {}
  for (const name of Object.keys(map)) sel[name] = !allowed || allowed.has(name)
  sel['全部'] = true
  rLegendSelected.value = sel
}, { immediate: true })

function rOnLegendChange(e) {
  rLegendSelected.value = { ...e.selected }
}

const rIsAllSelected = computed(() =>
  Object.values(rLegendSelected.value).every(v => v)
)

function rToggleAllLegend() {
  const next = !rIsAllSelected.value
  const sel = {}
  for (const k of Object.keys(rLegendSelected.value)) sel[k] = next
  rLegendSelected.value = sel
}

const rChartOption = computed(() => {
  const roundSet = new Set()
  for (const row of rRows.value) {
    if (row.lottery_round) roundSet.add(row.lottery_round)
  }
  const rounds = [...roundSet].sort((a, b) => a - b)

  if (rounds.length === 0) return { series: [] }

  const visibleRounds = rChartRange.value > 0 ? rounds.slice(0, rChartRange.value) : rounds

  const agg = {}
  const totalByRound = {}
  for (const row of rRows.value) {
    const round = row.lottery_round
    if (!round) continue
    if (!agg[row.member_name]) agg[row.member_name] = {}
    if (!agg[row.member_name][round]) agg[row.member_name][round] = { applied: 0, won: 0 }
    agg[row.member_name][round].applied += row.total_applied
    agg[row.member_name][round].won     += row.total_won
    if (!totalByRound[round]) totalByRound[round] = { applied: 0, won: 0 }
    totalByRound[round].applied += row.total_applied
    totalByRound[round].won     += row.total_won
  }

  const members = Object.keys(agg).sort((a, b) => memberOrderIndex(a) - memberOrderIndex(b))
  const xLabels = visibleRounds.map(r => formatRound(r))

  const winRate = (d) => d && d.applied ? parseFloat((d.won / d.applied * 100).toFixed(1)) : null

  const series = [
    ...members.map((member, i) => ({
      name: member,
      type: 'line',
      smooth: true,
      connectNulls: false,
      color: CHART_COLORS[i % CHART_COLORS.length],
      symbol: 'circle',
      symbolSize: 7,
      data: visibleRounds.map(r => winRate(agg[member][r])),
    })),
    {
      name: '全部',
      type: 'line',
      smooth: true,
      lineStyle: { width: 3, type: 'dashed' },
      color: '#333',
      symbol: 'diamond',
      symbolSize: 8,
      data: visibleRounds.map(r => winRate(totalByRound[r])),
    },
  ]

  const c = ct.value
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      formatter(params) {
        const idx = params[0].dataIndex
        let html = `<b>${xLabels[idx]}</b><br/>`
        params.forEach(p => {
          if (p.value !== null && p.value !== undefined)
            html += `${p.marker}${p.seriesName}：${p.value}%<br/>`
        })
        return html
      },
    },
    legend: { data: [...members, '全部'], bottom: 0, type: 'scroll', selected: rLegendSelected.value, textStyle: { color: c.text } },
    grid: { top: 16, right: 24, bottom: 56, left: 54 },
    xAxis: { type: 'category', data: xLabels, axisLabel: { color: c.text }, axisLine: { lineStyle: { color: c.line } } },
    yAxis: {
      type: 'value', min: 0, max: 100,
      axisLabel: { formatter: '{value}%', color: c.text },
      splitLine: { lineStyle: { type: 'dashed', color: c.line } },
    },
    series,
  }
})

// ══════════════════════════════════════════════════════════════
// 全握總表
// ══════════════════════════════════════════════════════════════

const fOverall          = ref({ total_applied: 0, total_won: 0 })
const fByType            = ref([])
const fContributorCount  = ref(0)
const fMemberStats       = ref([])
const fMemberList        = ref([])
const fVenueList         = ref([])
const fRegionStats       = ref([])

const fMemberFilterGroup  = ref('')
const fMemberFilterType   = ref('')
const fMemberFilterRegion = ref('')
const fMemberFilterVenue  = ref('')
const fMemberRowModes     = ref(['single', 'multi'])
const fRegionFilterGroup  = ref('')

const fOpenSections = ref(['type', 'region', 'member'])

const fDetailFilterGroup = ref('')
const fDetailMember      = ref('')
const fDetailActiveOnly  = ref(false)
const fDetailType        = ref('実体')
const fDetailRegion      = ref('')
const fDetailVenue       = ref('')
const fSelectedRounds    = ref([1])
const fDetailData        = ref([])
const fDetailLoading     = ref(false)

const fOverallRate = computed(() => {
  if (!fOverall.value.total_applied) return '0.0'
  return (fOverall.value.total_won / fOverall.value.total_applied * 100).toFixed(1)
})

const fDetailVenueOptions = computed(() =>
  fDetailRegion.value ? fVenueList.value.filter(v => REGION_VENUES[fDetailRegion.value]?.includes(v)) : fVenueList.value
)

const fMemberVenueOptions = computed(() =>
  fMemberFilterRegion.value ? fVenueList.value.filter(v => REGION_VENUES[fMemberFilterRegion.value]?.includes(v)) : fVenueList.value
)

const fDetailMemberOptions = computed(() => fMemberList.value.filter(m => {
  if (fDetailFilterGroup.value && m.group !== fDetailFilterGroup.value) return false
  if (fDetailActiveOnly.value && !(getMemberInfo(m.name).active ?? true)) return false
  return true
}))

const fFilteredMemberStats = computed(() => fMemberStats.value.filter(r => {
  const isMulti = r.member_name.includes('・')
  return isMulti ? fMemberRowModes.value.includes('multi') : fMemberRowModes.value.includes('single')
}))

const fDetailSessions       = computed(() => [...new Set(fDetailData.value.map(r => r.session))].sort())
const fSelectedRoundsSorted = computed(() => [...fSelectedRounds.value].sort((a, b) => a - b))

const fDetailRows = computed(() => {
  const map = {}
  fDetailData.value.forEach(r => {
    const key = `${r.single_number}:${r.member_name}:${r.venue}`
    if (!map[key]) {
      const partners = r.member_name.split('・').filter(n => n !== fDetailMember.value)
      map[key] = {
        single_number: r.single_number,
        single_name:   r.single_name,
        venue:         r.venue || '',
        partner:       partners.length > 0 ? partners.join('・') : '',
        cells: {},
      }
    }
    map[key].cells[`${r.session}:${r.lottery_round}`] = { applied: r.total_applied, won: r.total_won }
  })
  return Object.values(map).sort((a, b) =>
    a.single_number !== b.single_number
      ? a.single_number - b.single_number
      : a.venue.localeCompare(b.venue)
  )
})

async function fLoadStats() {
  const statsRes = await getGlobalFullOverallStats()
  fOverall.value = statsRes.data.overall ?? { total_applied: 0, total_won: 0 }
  fByType.value  = (statsRes.data.by_type ?? []).map(r => ({ ...r, win_rate_num: parseFloat(r.win_rate) }))
  fContributorCount.value = statsRes.data.contributor_count ?? 0
  fVenueList.value = [...new Set(fByType.value.map(r => r.venue).filter(v => v))]
  await fLoadRegionStats()
  await fLoadMemberStats()
  const nameGroupMap = new Map()
  fMemberStats.value.forEach(m => m.member_name.split('・').forEach(n => { n = n.trim(); if (n) nameGroupMap.set(n, m.group || '') }))
  fMemberList.value = sortMembersByGroupAndGen([...nameGroupMap.entries()].map(([name, group]) => ({ name, group })))
}

async function fLoadRegionStats() {
  const params = {}
  if (fRegionFilterGroup.value) params.group = fRegionFilterGroup.value
  const res = await getGlobalFullStatsByRegion(params)
  fRegionStats.value = (res.data ?? []).map(r => ({ ...r, win_rate_num: parseFloat(r.win_rate) }))
}

function fOnMemberGroupChange() {
  fLoadMemberStats()
}

function fOnMemberTypeChange() {
  if (fMemberFilterType.value === '線上') { fMemberFilterRegion.value = ''; fMemberFilterVenue.value = '' }
  fLoadMemberStats()
}

function fOnMemberRegionChange() {
  if (fMemberFilterVenue.value && !fMemberVenueOptions.value.includes(fMemberFilterVenue.value)) fMemberFilterVenue.value = ''
  fLoadMemberStats()
}

async function fLoadMemberStats() {
  const params = {}
  if (fMemberFilterGroup.value)  params.group      = fMemberFilterGroup.value
  if (fMemberFilterType.value)   params.event_type = fMemberFilterType.value
  if (fMemberFilterRegion.value) params.region     = fMemberFilterRegion.value
  if (fMemberFilterVenue.value)  params.venue      = fMemberFilterVenue.value
  const res = await getGlobalFullStatsByMember(params)
  fMemberStats.value = (res.data ?? []).map(r => ({ ...r, win_rate_num: parseFloat(r.win_rate) }))
}

function fOnDetailTypeChange() {
  if (fDetailType.value === '線上') { fDetailRegion.value = ''; fDetailVenue.value = '' }
  fLoadDetail()
}

function fOnDetailActiveOnlyChange() {
  if (fDetailMember.value && !fDetailMemberOptions.value.some(m => m.name === fDetailMember.value)) {
    fDetailMember.value = ''
    fLoadDetail()
  }
}

function fOnDetailGroupChange() {
  if (fDetailMember.value && !fDetailMemberOptions.value.some(m => m.name === fDetailMember.value)) {
    fDetailMember.value = ''
    fLoadDetail()
  }
}

function fOnDetailRegionChange() {
  if (fDetailVenue.value && !fDetailVenueOptions.value.includes(fDetailVenue.value)) fDetailVenue.value = ''
  fLoadDetail()
}

async function fLoadDetail() {
  if (!fDetailMember.value) { fDetailData.value = []; return }
  fDetailLoading.value = true
  try {
    const params = { member: fDetailMember.value }
    if (fDetailType.value)   params.event_type = fDetailType.value
    if (fDetailRegion.value) params.region = fDetailRegion.value
    if (fDetailVenue.value)  params.venue = fDetailVenue.value
    const res = await getGlobalFullDetailStats(params)
    fDetailData.value = res.data ?? []
  } finally {
    fDetailLoading.value = false
  }
}
</script>

<style scoped>
.page { background: #f5f7fa; min-height: 100vh; }
:deep(.el-table .cell) { white-space: nowrap; }

.quick-start-collapse { margin-bottom: 20px; }
.quick-start-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
}
.qs-step {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  flex: 1 1 220px;
}
.qs-num {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--color-primary);
  color: white;
  font-weight: bold;
  font-size: 13px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.qs-title { font-weight: 600; color: #333; margin-bottom: 4px; font-size: 13px; }
.qs-sub   { font-size: 12px; color: #888; line-height: 1.6; }
.qs-link  { display: inline-block; margin-top: 6px; font-size: 12px; color: var(--color-primary); }
.qs-dismiss {
  text-align: right;
  font-size: 12px;
  color: #999;
  cursor: pointer;
  margin-top: 16px;
}
.qs-dismiss:hover { color: var(--color-primary); }
html.dark .qs-title { color: #d4d8e3; }
html.dark .qs-sub   { color: #9aa3b5; }

:deep(.el-tabs__nav-wrap)  { margin-bottom: 4px; }
:deep(.el-tabs__item)      { font-weight: 600; }

:deep(.el-collapse) { border: none; background: transparent; }
:deep(.el-collapse-item) {
  margin-bottom: 12px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid #e5e7eb;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.07);
  background: white;
}
:deep(.el-collapse-item__header) {
  height: 52px;
  padding: 0 20px;
  font-size: 14px;
  font-weight: 600;
  color: #111827;
  background: white;
  border-bottom: 1px solid transparent;
}
:deep(.el-collapse-item.is-active .el-collapse-item__header) { border-bottom-color: #e5e7eb; }
:deep(.el-collapse-item__arrow) { color: #6b7280; }
:deep(.el-collapse-item__wrap) { background: white; border: none; }
:deep(.el-collapse-item__content) { padding: 16px 20px 20px; }

.collapse-title { font-weight: 600; font-size: 14px; }
.filters { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.detail-filters { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; margin-bottom: 16px; }
.detail-sub { font-size: 11px; color: #999; }
.partner-name { font-size: 12px; color: #6366f1; }
.text-muted { color: #bbb; }
.empty { text-align: center; color: #999; padding: 40px 0; }
.hint-text { font-size: 12px; color: #999; margin-top: 10px; line-height: 1.5; }

.stats-grid {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.stat-card {
  background: white;
  border-radius: 10px;
  padding: 16px 24px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.07);
  border: 1px solid #e5e7eb;
  min-width: 140px;
}
.stat-label { font-size: 13px; color: #888; margin-bottom: 6px; }
.stat-value { font-size: 24px; font-weight: bold; color: #222; }
.stat-value.highlight { color: #6366f1; }

/* 圖表共用 */
.rank-sub-title {
  font-size: 13px;
  font-weight: 600;
  color: #666;
  margin: 20px 0 10px;
}
.chart-range-btns {
  display: flex;
  gap: 6px;
  margin-bottom: 14px;
}
.range-btn {
  padding: 3px 12px;
  border: 1px solid #ddd;
  border-radius: 20px;
  background: white;
  font-size: 13px;
  cursor: pointer;
  color: #666;
  transition: all 0.15s;
}
.range-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }
.range-btn.active { background: var(--color-primary); border-color: var(--color-primary); color: white; }
.range-divider { color: #ddd; align-self: center; }
.chart-filters {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}
.chart-empty {
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #bbb;
  font-size: 14px;
}

/* 成員層 */
.member-list-header {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-bottom: 8px;
}
.member-filter-select {
  width: 260px;
}
.member-list { display: flex; flex-direction: column; gap: 12px; }

/* 團體層 */
.group-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  overflow: hidden;
}
.group-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}
.group-header:hover { background: #f5f5f5; }
.group-name { font-size: 18px; font-weight: bold; flex: 1; }
.group-body { padding: 0 16px 16px; display: flex; flex-direction: column; gap: 12px; }

.member-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  overflow: hidden;
}

.member-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}
.member-header:hover { background: #f5f5f5; }
.member-name { font-size: 18px; font-weight: bold; flex: 1; }
.member-summary { color: #666; font-size: 14px; }
.member-summary .rate,
.single-summary .rate { color: var(--color-primary); font-weight: bold; margin-left: 6px; }
.chevron { color: #bbb; font-size: 11px; }

.member-body { padding: 0 16px 16px; display: flex; flex-direction: column; gap: 8px; }

/* 單曲層 */
.single-card {
  border: 1px solid #eee;
  border-radius: 8px;
  overflow: hidden;
}

.single-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  cursor: pointer;
  user-select: none;
  background: #fafafa;
  transition: background 0.15s;
}
.single-header:hover { background: #f0f0f0; }
.single-name { font-size: 15px; font-weight: 600; color: var(--color-primary); flex: 1; }
.single-summary { color: #888; font-size: 13px; }

.single-body { padding: 0 16px 16px; }

/* 次數層（手風琴） */
.round-card {
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  overflow: hidden;
  margin-top: 8px;
}

.round-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 14px;
  cursor: pointer;
  user-select: none;
  background: #f5f5f5;
  border-left: 3px solid var(--color-primary);
  transition: background 0.15s;
}
.round-header:hover { background: #ececec; }

.round-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-primary);
  flex: 1;
}

.round-summary {
  color: #888;
  font-size: 12px;
}
.round-summary .rate { color: var(--color-primary); font-weight: bold; margin-left: 4px; }

/* 表格 */
.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.detail-table th {
  background: #f7f7f7;
  padding: 7px 12px;
  text-align: left;
  color: #888;
  font-weight: 500;
}
.detail-table td {
  padding: 7px 12px;
  border-bottom: 1px solid #f0f0f0;
}
.detail-table tr:last-child td { border-bottom: none; }

.rate { font-weight: bold; }
.rate.high { color: #52c41a; }
.rate.mid  { color: #faad14; }
.rate.low  { color: #ff4d4f; }

/* ── 深色模式 ── */
html.dark .stat-card  { background: #1e2030; box-shadow: 0 2px 12px rgba(0,0,0,0.4); border-color: #2e3450; }
html.dark .stat-label { color: #9aa3b5; }
html.dark .stat-value { color: #e8eaf0; }

html.dark .chart-empty { color: #6b7490; }
html.dark .rank-sub-title { color: #b8bfcc; }

html.dark .range-btn         { background: #252840; border-color: #3a3f5c; color: #b8bfcc; }
html.dark .range-btn:hover   { border-color: var(--color-primary); color: var(--color-primary); }
html.dark .range-btn.active  { background: var(--color-primary); border-color: var(--color-primary); color: white; }
html.dark .range-divider     { color: #3a3f5c; }

html.dark .group-card            { background: #1e2030; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
html.dark .group-header:hover    { background: #252840; }

html.dark .member-card           { background: #1e2030; box-shadow: 0 2px 12px rgba(0,0,0,0.4); }
html.dark .member-header:hover   { background: #252840; }
html.dark .member-name           { color: #e8eaf0; }
html.dark .member-summary        { color: #9aa3b5; }
html.dark .chevron               { color: #4a5270; }

html.dark .single-card           { border-color: #2e3450; }
html.dark .single-header         { background: #252840; }
html.dark .single-header:hover   { background: #2c3154; }
html.dark .single-summary        { color: #9aa3b5; }

html.dark .round-card            { border-color: #2e3450; }
html.dark .round-header          { background: #1a1f3a; }
html.dark .round-header:hover    { background: #20264a; }
html.dark .round-summary         { color: #9aa3b5; }

html.dark .detail-table th       { background: #252840; color: #9aa3b5; }
html.dark .detail-table td       { border-bottom-color: #2e3450; color: #d4d8e3; }
</style>
