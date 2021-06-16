<template>
  <page-default>

    <section v-if="!loading" class="app-section">
      <div class="app-section-wrap app-boxed app-boxed-xl q-pl-md q-pr-md q-pt-xl q-pb-lg">
        <div v-if="serviceByName.item" class="row no-wrap items-center app-title">
          <div class="app-title-label" style="font-size: 26px">{{ serviceByName.item.name }}</div>
        </div>
      </div>
    </section>

    <section class="app-section">
      <div class="app-section-wrap app-boxed app-boxed-xl q-pl-md q-pr-md q-pt-lg q-pb-lg">
        <div v-if="!loading" class="row items-start q-col-gutter-md">

          <div v-if="serviceByName.item" class="col-12 col-md-4 q-mb-lg path-block">
            <div class="row no-wrap items-center q-mb-lg app-title">
              <q-icon name="eva-info"></q-icon>
              <div class="app-title-label">Service Details</div>
            </div>
            <div class="row items-start q-col-gutter-lg">
              <div class="col-12">
                <div class="row items-start q-col-gutter-md">
                  <div class="col-12">
                    <panel-service-details dense :data="serviceByName.item" />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="serviceByName.item.loadBalancer && serviceByName.item.loadBalancer.healthCheck" class="col-12 col-md-4 q-mb-lg path-block">
            <div class="row no-wrap items-center q-mb-lg app-title">
              <q-icon name="eva-shield"></q-icon>
              <div class="app-title-label">Health Check</div>
            </div>
            <div class="row items-start q-col-gutter-lg">
              <div class="col-12">
                <div class="row items-start q-col-gutter-md">
                  <div class="col-12">
                    <panel-health-check dense :data="serviceByName.item.loadBalancer.healthCheck"/>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="serviceByName.item.loadBalancer" class="col-12 col-md-4 q-mb-lg path-block">
            <div class="row no-wrap items-center q-mb-lg app-title">
              <q-icon name="eva-globe-outline"></q-icon>
              <div class="app-title-label">Servers</div>
            </div>
            <div class="row items-start q-col-gutter-lg">
              <div class="col-12">
                <div class="row items-start q-col-gutter-md">
                  <div class="col-12">
                    <panel-servers dense :data="serviceByName.item" :hasStatus="serviceByName.item.serverStatus"/>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="serviceByName.item.weighted && serviceByName.item.weighted.services" class="col-12 col-md-4 q-mb-lg path-block">
            <div class="row no-wrap items-center q-mb-lg app-title">
              <q-icon name="eva-globe-outline"></q-icon>
              <div class="app-title-label">Services</div>
            </div>
            <div class="row items-start q-col-gutter-lg">
              <div class="col-12">
                <div class="row items-start q-col-gutter-md">
                  <div class="col-12">
                    <panel-weighted-services dense :data="serviceByName.item"/>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="serviceByName.item.mirroring && serviceByName.item.mirroring.mirrors" class="col-12 col-md-4 q-mb-lg path-block">
            <div class="row no-wrap items-center q-mb-lg app-title">
              <q-icon name="eva-globe-outline"></q-icon>
              <div class="app-title-label">Mirror Services</div>
            </div>
            <div class="row items-start q-col-gutter-lg">
              <div class="col-12">
                <div class="row items-start q-col-gutter-md">
                  <div class="col-12">
                    <panel-mirroring-services dense :data="serviceByName.item"/>
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
import PanelServiceDetails from '../../components/_commons/PanelServiceDetails'
import PanelHealthCheck from '../../components/_commons/PanelHealthCheck'
import PanelServers from '../../components/_commons/PanelServers'
import MainTable from '../../components/_commons/MainTable'
import PanelWeightedServices from '../../components/_commons/PanelWeightedServices'
import PanelMirroringServices from '../../components/_commons/PanelMirroringServices'

export default {
  name: 'PageServiceDetail',
  props: ['name', 'type'],
  mixins: [GetTablePropsMixin],
  components: {
    PanelMirroringServices,
    PanelWeightedServices,
    PageDefault,
    SkeletonBox,
    PanelServiceDetails,
    PanelHealthCheck,
    PanelServers,
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
    ...mapGetters('http', { http_serviceByName: 'serviceByName' }),
    ...mapGetters('tcp', { tcp_serviceByName: 'serviceByName' }),
    ...mapGetters('udp', { udp_serviceByName: 'serviceByName' }),
    protocol () {
      return this.$route.meta.protocol
    },
    serviceByName () {
      return this[`${this.protocol}_serviceByName`]
    },
    getServiceByName () {
      return this[`${this.protocol}_getServiceByName`]
    },
    getRouterByName () {
      return this[`${this.protocol}_getRouterByName`]
    }
  },
  methods: {
    ...mapActions('http', { http_getServiceByName: 'getServiceByName', http_getRouterByName: 'getRouterByName' }),
    ...mapActions('tcp', { tcp_getServiceByName: 'getServiceByName', tcp_getRouterByName: 'getRouterByName' }),
    ...mapActions('udp', { udp_getServiceByName: 'getServiceByName', udp_getRouterByName: 'getRouterByName' }),
    refreshAll () {
      if (this.serviceByName.loading) {
        return
      }
      this.onGetAll()
    },
    onGetAll () {
      this.getServiceByName(this.name)
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
          console.log('Error -> service/byName', error)
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
    this.$store.commit('http/getServiceByNameClear')
    this.$store.commit('tcp/getServiceByNameClear')
    this.$store.commit('udp/getServiceByNameClear')
  }
}
</script>

<style scoped lang="scss">
  @import "../../css/sass/variables";

</style>
