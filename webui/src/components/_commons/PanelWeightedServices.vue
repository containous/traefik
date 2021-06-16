<template>
  <q-card flat bordered v-bind:class="['panel-services', {'panel-services-dense':isDense}]">
    <q-scroll-area :thumb-style="appThumbStyle" style="height:100%;">
      <q-card-section>
        <div class="row items-start no-wrap">
          <div class="col-7">
            <div class="text-subtitle2 text-table">Name</div>
          </div>
          <div class="col-3">
            <div class="text-subtitle2 text-table">Weight</div>
          </div>
          <div class="col-4">
            <div class="text-subtitle2 text-table">Provider</div>
          </div>
        </div>
      </q-card-section>
      <q-separator />
      <div v-for="(service, index) in data.weighted.services" :key="index">
        <q-card-section>
          <div class="row items-center no-wrap">
            <div class="col-7">
              <q-chip
                dense
                class="app-chip app-chip-rule app-chip-overflow">
                {{ service.name }}
                <q-tooltip>{{service.name}}</q-tooltip>
              </q-chip>
            </div>
            <div class="col-3">
              {{ service.weight }}
            </div>
            <div class="col-4">
              <q-avatar>
                <q-icon :name="`img:${getProviderLogoPath(service)}`" />
              </q-avatar>
            </div>
          </div>
        </q-card-section>
        <q-separator />
      </div>
    </q-scroll-area>
  </q-card>
</template>

<script>

export default {
  name: 'PanelWeightedServices',
  props: ['data', 'dense'],
  components: {},
  computed: {
    isDense () {
      return this.dense !== undefined
    }
  },
  methods: {
    getProvider (service) {
      const words = service.name.split('@')
      if (words.length === 2) {
        return words[1]
      }

      return this.data.provider
    },
    getProviderLogoPath (service) {
      const provider = this.getProvider(service)
      const name = provider.name.toLowerCase()

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

  .panel-services {
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
  }

</style>
