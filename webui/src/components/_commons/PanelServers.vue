<template>
  <q-card flat bordered v-bind:class="['panel-servers', {'panel-servers-dense':isDense}]">
    <q-scroll-area v-if="data.loadBalancer.servers" :thumb-style="appThumbStyle" style="height:100%;">
      <q-card-section>
        <div class="row items-start no-wrap">
          <div class="col-3" v-if="showStatus">
            <div class="text-subtitle2 text-table">Status</div>
          </div>
          <div class="col-9">
            <div class="text-subtitle2 text-table">URL</div>
          </div>
        </div>
      </q-card-section>
      <q-separator />
      <div v-for="(server, index) in data.loadBalancer.servers" :key="index">
        <q-card-section>
          <div class="row items-center no-wrap">
            <div class="col-3" v-if="showStatus">
              <div class="block-right-text">
                <avatar-state v-if="data.serverStatus" :state="data.serverStatus[server.url || server.address] | status "/>
                <avatar-state v-if="!data.serverStatus" :state="'DOWN' | status"/>
              </div>
            </div>
            <div class="col-9">
              <q-chip
                dense
                class="app-chip app-chip-rule">
                {{ server.url || server.address}}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <q-separator />
      </div>
    </q-scroll-area>
      <q-card-section v-else style="height: 100%">
        <div class="row items-center" style="height: 100%">
          <div class="col-12">
            <div class="block-empty"></div>
            <div class="q-pb-lg block-empty-logo">
              <img v-if="$q.dark.isActive" alt="empty" src="~assets/middlewares-empty-dark.svg">
              <img v-else alt="empty" src="~assets/middlewares-empty.svg">
            </div>
            <div class="block-empty-label">There is no<br>Server available</div>
          </div>
        </div>
      </q-card-section>
  </q-card>
</template>

<script>
import AvatarState from './AvatarState'

export default {
  name: 'PanelServers',
  props: ['data', 'dense', 'hasStatus'],
  components: {
    AvatarState
  },
  computed: {
    isDense () {
      return this.dense !== undefined
    },
    showStatus () {
      return this.hasStatus !== undefined
    }
  },
  filters: {
    status (value) {
      if (value === 'UP') {
        return 'positive'
      }
      return 'negative'
    }
  }
}
</script>

<style scoped lang="scss">
  @import "../../css/sass/variables";

  .panel-servers {
    height: 600px;
    &-dense{
      height: 400px;
    }
    .q-card__section {
      padding: 12px 24px;
      + .q-card__section {
        padding-top: 0;
      }
    }

    .block-right-text{
      height: 32px;
      line-height: 32px;
      .q-avatar{
        float: left;
      }
      &-label{
        font-size: 14px;
        font-weight: 600;
        color: $app-text-grey;
        float: left;
        margin-left: 10px;
        text-transform: capitalize;
        &-enabled {
          color: $positive;
        }
        &-disabled {
          color: $negative;
        }
        &-warning {
          color: $warning;
        }
      }
    }

    .text-subtitle2 {
      font-size: 11px;
      color: $app-text-grey;
      line-height: 16px;
      margin-bottom: 4px;
      text-align: left;
      letter-spacing: 2px;
      font-weight: 600;
      text-transform: uppercase;
    }

    .text-table {
      font-size: 14px;
      font-weight: 700;
      letter-spacing: normal;
      text-transform: none;
    }

    .block-empty {
      &-logo {
         text-align: center;
       }
      &-label {
         font-size: 20px;
         font-weight: 700;
         color: #b8b8b8;
         text-align: center;
         line-height: 1.2;
       }
    }
  }

</style>
