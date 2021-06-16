<template>
  <div class="table-wrapper">
    <q-infinite-scroll @load="handleLoadMore" :offset="250" ref="scroller">
      <q-markup-table>
        <thead>
          <tr class="table-header">
            <th
              v-for="column in columns"
              v-bind:class="`text-${column.align}`"
              v-bind:key="column.name">
              {{ column.label }}
            </th>
          </tr>
        </thead>
        <tfoot v-if="!data || !data.length">
          <tr>
            <td colspan="100%">
              <q-icon name="warning" style="font-size: 1.5rem"/> No data available
            </td>
          </tr>
        </tfoot>
        <tbody>
          <tr v-for="row in data" :key="row.name" class="cursor-pointer" @click="onRowClick(row)">
            <template v-for="column in columns">
              <td :key="column.name" v-if="getColumn(column.name).component" v-bind:class="`text-${getColumn(column.name).align}`">
                <component
                  v-bind:is="getColumn(column.name).component"
                  v-bind="getColumn(column.name).fieldToProps(row)"
                >
                  <template v-if="getColumn(column.name).content">
                    {{ getColumn(column.name).content(row) }}
                  </template>
                </component>
              </td>
              <td
                :key="column.name"
                v-if="!getColumn(column.name).component"
                v-bind:class="`text-${getColumn(column.name).align}`"
                v-bind="getColumn(column.name).fieldToProps(row)"
              >
                 <span>
                  {{getColumn(column.name).content ? getColumn(column.name).content(row) : row[column.name]}}
                </span>
              </td>
            </template>
          </tr>
        </tbody>
      </q-markup-table>
      <template v-slot:loading v-if="loading">
        <div class="row justify-center q-my-md">
          <q-spinner-dots color="app-grey" size="40px" />
        </div>
      </template>
    </q-infinite-scroll>
    <q-page-scroller position="bottom" :scroll-offset="150" class="back-to-top" v-if="endReached">
      <q-btn color="primary" small>
        Back to top
      </q-btn>
    </q-page-scroller>
  </div>
</template>

<script>
import { QMarkupTable, QInfiniteScroll, QSpinnerDots, QPageScroller } from 'quasar'

export default {
  name: 'MainTable',
  props: ['data', 'columns', 'loading', 'onLoadMore', 'endReached', 'onRowClick'],
  components: {
    QMarkupTable,
    QInfiniteScroll,
    QSpinnerDots,
    QPageScroller
  },
  methods: {
    getColumn (columnName) {
      return this.columns.find(c => c.name === columnName) || {}
    },
    handleLoadMore (index, done) {
      this.onLoadMore({ page: index })
        .then(() => done())
        .catch(() => done(true))
    }
  }
}
</script>

<style scoped lang="scss">
  @import "../../css/sass/variables";

  .table-wrapper {
    /deep/ .q-table__container{
      border-radius: 8px;
      .q-table {
        .table-header {
          th {
            font-size: 14px;
            font-weight: 700;
          }
        }
        tbody {
          tr:hover {
            background: rgba( $accent, 0.04 );
          }
        }
      }
      .q-table__bottom {
        > .q-table__control {
          &:nth-last-child(2) {
            display: none;
          }
          &:nth-last-child(1) {
            .q-table__bottom-item {
              display: none;
            }
          }
        }
      }
    }

    .back-to-top {
      margin: 16px 0;
    }
  }

  .servers-label {
    font-size: 14px;
    font-weight: 600;
  }
</style>
