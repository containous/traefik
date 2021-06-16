<template>
  <page-default>

    <section v-if="!loading" class="app-section">
      <div class="app-section-wrap app-boxed app-boxed-xl q-pl-md q-pr-md q-pt-xl q-pb-sm">
        <div v-if="middlewareByName.item" class="row no-wrap items-center app-title">
          <div class="app-title-label" style="font-size: 26px">{{ middlewareByName.item.name }}</div>
        </div>
      </div>
    </section>

    <section class="app-section">
      <div class="app-section-wrap app-boxed app-boxed-xl q-pl-md q-pr-md q-pt-sm q-pb-lg">
        <div v-if="!loading" class="row items-start q-col-gutter-md">

          <div v-if="middlewareByName.item" class="col-12 col-md-4 q-mb-lg path-block">
            <div class="row items-start q-col-gutter-lg">
              <div class="col-12">
                <div class="row items-start q-col-gutter-md">
                  <div class="col-12">
                    <panel-middlewares dense :data="[middlewareByName.item]" />
                  </div>
                </div>
              </div>
            </div>
          </div>

        </div>
        <div v-else class="row items-start q-mt-xl">
          <div class="col-12">
            <p v-for="n in 4" :key="n" class="flex">
              <SkeletonBox :min-width="15" :max-width="15" style="margin-right: 2%"/> <SkeletonBox :min-width="50" :max-width="83"/>
            </p>
          </div>
        </div>
      </div>
    </section>

    <section v-if="!loading && allRouters.length" class="app-section">
      <div class="app-section-wrap app-boxed app-boxed-xl q-pl-md q-pr-md q-pt-lg q-pb-xl">
        <div class="row no-wrap items-center q-mb-lg app-title">
          <div class="app-title-label">Used by Routers</div>
        </div>
        <div class="row items-center q-col-gutter-lg">
          <div class="col-12">
            <main-table
              :data="allRouters"
              v-bind="getTableProps({ type: `${protocol}-routers` })"
              :request="()=>{}"
              :loading="routersLoading"
              :pagination.sync="routersPagination"
              :filter="routersFilter"
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
import PageDefault from '../../components/_commons/PageDefault'
import SkeletonBox from '../../components/_commons/SkeletonBox'
import PanelMiddlewares from '../../components/_commons/PanelMiddlewares'
import MainTable from '../../components/_commons/MainTable'

export default {
  name: 'PageMiddlewareDetail',
  props: ['name', 'type'],
  mixins: [GetTablePropsMixin],
  components: {
    PageDefault,
    SkeletonBox,
    PanelMiddlewares,
    MainTable
  },
  data () {
    return {
      loading: true,
      timeOutGetAll: null,
      allRouters: [],
      routersLoading: true,
      routersFilter: '',
      routersStatus: '',
      routersPagination: {
        sortBy: '',
        descending: true,
        page: 1,
        rowsPerPage: 1000,
        rowsNumber: 0
      }
    }
  },
  computed: {
    ...mapGetters('http', { http_middlewareByName: 'middlewareByName' }),
    ...mapGetters('tcp', { tcp_middlewareByName: 'middlewareByName' }),
    protocol () {
      return this.$route.meta.protocol
    },
    middlewareByName () {
      return this[`${this.protocol}_middlewareByName`]
    },
    getMiddlewareByName () {
      return this[`${this.protocol}_getMiddlewareByName`]
    },
    getRouterByName () {
      return this[`${this.protocol}_getRouterByName`]
    }
  },
  methods: {
    ...mapActions('http', { http_getMiddlewareByName: 'getMiddlewareByName', http_getRouterByName: 'getRouterByName' }),
    ...mapActions('tcp', { tcp_getMiddlewareByName: 'getMiddlewareByName', tcp_getRouterByName: 'getRouterByName' }),
    refreshAll () {
      if (this.middlewareByName.loading) {
        return
      }
      this.onGetAll()
    },
    onGetAll () {
      this.getMiddlewareByName(this.name)
        .then(body => {
          if (!body) {
            this.loading = false
            return
          }
          // Get routers
          if (body.usedBy) {
            for (const router in body.usedBy) {
              if (body.usedBy.hasOwnProperty(router)) {
                this.getRouterByName(body.usedBy[router])
                  .then(body => {
                    if (body) {
                      this.routersLoading = false
                      this.allRouters.push(body)
                    }
                  })
                  .catch(error => {
                    console.log('Error -> routers/byName', error)
                  })
              }
            }
          }
          clearTimeout(this.timeOutGetAll)
          this.timeOutGetAll = setTimeout(() => {
            this.loading = false
          }, 300)
        })
        .catch(error => {
          console.log('Error -> middleware/byName', error)
        })
    }
  },
  created () {
    this.refreshAll()
  },
  mounted () {

  },
  beforeDestroy () {
    clearInterval(this.timeOutGetAll)
    this.$store.commit('http/getMiddlewareByNameClear')
    this.$store.commit('tcp/getMiddlewareByNameClear')
  }
}
</script>

<style scoped lang="scss">
  @import "../../css/sass/variables";

</style>
