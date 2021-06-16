<template>
  <page-default>

    <section class="app-section">
      <div class="app-section-wrap app-boxed app-boxed-xl q-pl-md q-pr-md q-pt-xl q-pb-xl">
        <div class="row no-wrap items-center q-mb-lg">
          <tool-bar-table :status.sync="status" :filter.sync="filter"/>
        </div>
        <div class="row items-center q-col-gutter-lg">
          <div class="col-12">
            <main-table
              ref="mainTable"
              v-bind="getTableProps({ type: 'tcp-routers' })"
              :data="allRouters.items"
              :onLoadMore="handleLoadMore"
              :endReached="allRouters.endReached"
              :loading="allRouters.loading"
            />
          </div>
        </div>
      </div>
    </section>

  </page-default>
</template>

<script>
import { mapActions, mapGetters } from 'vuex'
import GetTablePropsMixin from '../../_mixins/GetTableProps'
import PaginationMixin from '../../_mixins/Pagination'
import PageDefault from '../../components/_commons/PageDefault'
import ToolBarTable from '../../components/_commons/ToolBarTable'
import MainTable from '../../components/_commons/MainTable'

export default {
  name: 'PageTCPRouters',
  mixins: [
    GetTablePropsMixin,
    PaginationMixin({
      fetchMethod: 'getAllRoutersWithParams',
      scrollerRef: 'mainTable.$refs.scroller',
      pollingIntervalTime: 5000
    })
  ],
  components: {
    PageDefault,
    ToolBarTable,
    MainTable
  },
  data () {
    return {
      filter: '',
      status: ''
    }
  },
  computed: {
    ...mapGetters('tcp', { allRouters: 'allRouters' })
  },
  methods: {
    ...mapActions('tcp', { getAllRouters: 'getAllRouters' }),
    getAllRoutersWithParams (params) {
      return this.getAllRouters({
        query: this.filter,
        status: this.status,
        ...params
      })
    },
    refreshAll () {
      if (this.allRouters.loading) {
        return
      }

      this.initFetch()
    },
    handleLoadMore ({ page = 1 } = {}) {
      return this.fetchMore({ page })
    }
  },
  watch: {
    'status' () {
      this.refreshAll()
    },
    'filter' () {
      this.refreshAll()
    }
  },
  beforeDestroy () {
    this.$store.commit('tcp/getAllRoutersClear')
  }
}
</script>

<style scoped lang="scss">

</style>
