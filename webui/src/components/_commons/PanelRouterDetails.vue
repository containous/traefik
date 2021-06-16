<template>
  <q-card flat bordered v-bind:class="['panel-router-details']">
    <q-scroll-area :thumb-style="appThumbStyle" style="height:100%;">
      <q-card-section>
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">STATUS</div>
            <div class="block-right-text">
              <avatar-state :state="data.status | status "/>
              <div v-bind:class="['block-right-text-label', `block-right-text-label-${data.status}`]">{{data.status | statusLabel}}</div>
            </div>
          </div>
          <div class="col">
            <div class="text-subtitle2">PROVIDER</div>
            <div class="block-right-text">
              <q-avatar class="provider-logo">
                <q-icon :name="`img:${getProviderLogoPath}`" />
              </q-avatar>
              <div class="block-right-text-label">{{data.provider}}</div>
            </div>
          </div>
        </div>
      </q-card-section>
      <q-card-section v-if="data.rule">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">RULE</div>
            <q-chip
              dense
              class="app-chip app-chip-wrap app-chip-rule">
              {{ data.rule }}
            </q-chip>
          </div>
        </div>
      </q-card-section>
      <q-card-section v-if="data.name">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">NAME</div>
            <q-chip
              dense
              class="app-chip app-chip-wrap app-chip-name">
              {{ data.name }}
            </q-chip>
          </div>
        </div>
      </q-card-section>
      <q-card-section v-if="data.using">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">ENTRYPOINTS</div>
            <q-chip
              v-for="(entryPoint, index) in data.using" :key="index"
              dense
              class="app-chip app-chip-entry-points">
              {{ entryPoint }}
            </q-chip>
          </div>
        </div>
      </q-card-section>
      <q-card-section v-if="data.service">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">SERVICE</div>
            <q-chip
              dense
              clickable
              @click.native="$router.push({ path: `/${protocol}/services/${getServiceId()}`})"
              class="app-chip app-chip-wrap app-chip-service app-chip-overflow">
              {{ data.service }}
              <q-tooltip>{{ data.service }}</q-tooltip>
            </q-chip>
          </div>
        </div>
      </q-card-section>
      <q-card-section v-if="data.error">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">ERRORS</div>
            <q-chip
              v-for="(errorMsg, index) in data.error" :key="index"
              class="app-chip app-chip-error">
              {{ errorMsg }}
            </q-chip>
          </div>
        </div>
      </q-card-section>
    </q-scroll-area>
  </q-card>
</template>

<script>
import AvatarState from './AvatarState'

export default {
  name: 'PanelRouterDetails',
  props: ['data', 'protocol'],
  components: {
    AvatarState
  },
  methods: {
    getServiceId () {
      const words = this.data.service.split('@')
      if (words.length === 2) {
        return this.data.service
      }

      return `${this.data.service}@${this.data.provider}`
    }
  },
  filters: {
    status (value) {
      if (value === 'enabled') {
        return 'positive'
      }
      if (value === 'disabled') {
        return 'negative'
      }
      return value
    },
    statusLabel (value) {
      if (value === 'enabled') {
        return 'success'
      }
      if (value === 'disabled') {
        return 'error'
      }
      return value
    }
  },
  computed: {
    getProviderLogoPath () {
      const name = this.data.provider.toLowerCase()

      if (name.includes('plugin-')) {
        return 'statics/providers/plugin.svg'
      }

      return `statics/providers/${name}.svg`
    }
  }
}
</script>

<style scoped lang="scss">
  @import "../../css/sass/variables";

  .panel-router-details {
    height: 600px;
    .q-card__section {
      padding: 24px;
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

    .app-chip {
      &-error {
        display: flex;
        height: 100%;
        flex-wrap: wrap;
        border-width: 0;
        margin-bottom: 8px;
        /deep/ .q-chip__content{
          white-space: normal;
        }
      }
    }

    .provider-logo {
      width: 32px;
      height: 32px;
      img {
        width: 100%;
        height: 100%;
      }
    }
  }

</style>
