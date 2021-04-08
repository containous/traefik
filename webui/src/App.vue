<template>
  <div id="q-app">
    <router-view />
    <platform-panel
      v-if="pilotEnabled" />
  </div>
</template>

<script>
import { APP } from './_helpers/APP'
import PlatformPanel from './components/platform/PlatformPanel'
import { mapGetters } from 'vuex'

export default {
  name: 'App',
  components: {
    PlatformPanel
  },
  computed: {
    ...mapGetters('core', { coreVersion: 'version' }),
    pilotEnabled () {
      return this.coreVersion.pilotEnabled
    }
  },
  beforeCreate () {
    // Set vue instance
    APP.vue = () => this.$root

    // debug
    console.log('Quasar -> ', this.$q.version)

    this.$q.dark.set(localStorage.getItem('traefik-dark') === 'true')
  },
  watch: {
    '$q.dark.isActive' (val) {
      localStorage.setItem('traefik-dark', val)
    }
  }
}
</script>

<style>
</style>
