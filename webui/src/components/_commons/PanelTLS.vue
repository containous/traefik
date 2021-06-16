<template>
  <q-card flat bordered v-bind:class="['panel-tls']">
    <q-scroll-area v-if="data" :thumb-style="appThumbStyle" style="height:100%;">
      <q-card-section v-if="data">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">TLS</div>
            <boolean-state :value="!!data"/>
          </div>
        </div>
      </q-card-section>
      <q-card-section v-if="data.options">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">OPTIONS</div>
            <q-chip
              dense
              class="app-chip app-chip-options">
              {{ data.options }}
            </q-chip>
          </div>
        </div>
      </q-card-section>
      <q-card-section v-if="protocol === 'tcp'">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">PASSTHROUGH</div>
            <boolean-state :value="data.passthrough"></boolean-state>
          </div>
        </div>
      </q-card-section>
      <q-card-section v-if="data.certResolver">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">CERTIFICATE RESOLVER</div>
            <q-chip
              dense
              class="app-chip app-chip-service">
              {{ data.certResolver }}
            </q-chip>
          </div>
        </div>
      </q-card-section>
      <q-card-section v-if="data.domains">
        <div class="row items-start no-wrap">
          <div class="col">
            <div class="text-subtitle2">DOMAINS</div>
            <div v-for="(domain, key) in data.domains" :key="key" class="flex">
              <q-chip
                dense
                class="app-chip app-chip-rule">
                {{ domain.main }}
              </q-chip>
              <q-chip
                v-for="(domain, key) in domain.sans" :key="key"
                dense
                class="app-chip app-chip-entry-points">
                {{ domain }}
              </q-chip>
            </div>
          </div>
        </div>
      </q-card-section>
    </q-scroll-area>
    <q-card-section v-else style="height: 100%">
      <div class="row items-center" style="height: 100%">
        <div class="col-12">
          <div class="block-empty"></div>
          <div class="q-pb-lg block-empty-logo">
            <img v-if="$q.dark.isActive" alt="empty" src="~assets/middlewares-empty-dark.svg">
            <img v-else alt="empty" src="~assets/middlewares-empty.svg">
          </div>
          <div class="block-empty-label">There is no<br>TLS configured</div>
        </div>
      </div>
    </q-card-section>
  </q-card>
</template>

<script>
import BooleanState from './BooleanState'

export default {
  name: 'PanelTLS',
  components: {
    BooleanState
  },
  props: ['data', 'protocol']
}
</script>

<style scoped lang="scss">
  @import "../../css/sass/variables";

  .panel-tls {
    height: 600px;
    .q-card__section {
      padding: 24px;
      + .q-card__section {
        padding-top: 0;
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
      &-entry-points {
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
