<template>
  <q-toolbar class="row no-wrap items-center">
    <q-tabs align="left" inline-label indicator-color="transparent" stretch>
      <q-route-tab :to="`/${protocol}/routers`" no-caps :label="`${protocolLabel} Routers`">
        <q-badge v-if="routerTotal !== 0" align="middle" :label="routerTotal" class="q-ml-sm"/>
      </q-route-tab>
      <q-route-tab :to="`/${protocol}/services`" no-caps :label="`${protocolLabel} Services`">
        <q-badge v-if="servicesTotal !== 0" align="middle" :label="servicesTotal" class="q-ml-sm"/>
      </q-route-tab>
      <q-route-tab v-if="protocol !== 'udp'" :to="`/${protocol}/middlewares`" no-caps :label="`${protocolLabel} Middlewares`">
        <q-badge v-if="middlewaresTotal !== 0" align="middle" :label="middlewaresTotal" class="q-ml-sm"/>
      </q-route-tab>
    </q-tabs>
  </q-toolbar>
</template>

<script>
import { mapActions, mapGetters } from 'vuex'

export default {
  name: 'ToolBar',
  data () {
    return {
      loadingOverview: true,
      intervalRefresh: null,
      intervalRefreshTime: 5000
    }
  },
  computed: {
    ...mapGetters('core', { overviewAll: 'allOverview' }),
    protocol () {
      return this.$route.meta.protocol
    },
    protocolLabel () {
      return this.protocol.toUpperCase()
    },
    routerTotal () {
      const data = this.overviewAll.items && this.overviewAll.items[`${this.protocol}`]
      return (data && data.routers && data.routers.total) || 0
    },
    servicesTotal () {
      const data = this.overviewAll.items && this.overviewAll.items[`${this.protocol}`]
      return (data && data.services && data.services.total) || 0
    },
    middlewaresTotal () {
      const data = this.overviewAll.items && this.overviewAll.items[`${this.protocol}`]
      return (data && data.middlewares && data.middlewares.total) || 0
    }
  },
  methods: {
    ...mapActions('core', { getOverview: 'getOverview' }),
    refreshAll () {
      this.onGetAll()
    },
    onGetAll () {
      this.getOverview()
        .then(body => {
          console.log('Success -> toolbar/overview', body)
          if (!body) {
            this.loadingOverview = false
          }
        })
        .catch(error => {
          console.log('Error -> toolbar/overview', error)
        })
    }
  },
  created () {
    this.refreshAll()
    this.intervalRefresh = setInterval(this.onGetAll, this.intervalRefreshTime)
  },
  beforeDestroy () {
    clearInterval(this.intervalRefresh)
    this.$store.commit('core/getOverviewClear')
  }
}
</script>

<style scoped lang="scss">
  @import "../../css/sass/variables";

  .q-toolbar {
    min-height: 48px;
    padding: 0;
  }

  .body--dark .q-toolbar {
    background-color: #0e204c;
  }

  .q-tabs {
    /deep/ .q-tabs__content {
      .q-tab__label {
        color: $app-text-grey;
        font-size: 16px;
        font-weight: 700;
      }
      .q-badge {
        font-size: 13px;
        font-weight: 700;
        border-radius: 12px;
        text-align: center;
        align-items: center;
        justify-content: center;
        min-width: 30px;
        padding: 6px;
        color: $app-text-grey;
        background-color: rgba( $app-text-grey, .1 );
      }
      .q-tab--active {
        .q-tab__label {
          color: $accent;
        }
        .q-badge {
          color: $accent;
          background-color: rgba( $accent, .1 );
        }
      }
    }
  }
</style>
