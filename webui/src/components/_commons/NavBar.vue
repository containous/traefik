<template>
  <q-header class="shadow-1">
    <section class="app-section bg-primary text-white">
      <div class="app-section-wrap app-boxed app-boxed-xl">
        <q-toolbar class="row no-wrap items-center">
          <div class="q-pr-md logo">
            <img alt="logo" src="~assets/logo.svg">
            <q-btn v-if="version" type="a" href="https://github.com/traefik/traefik/" target="_blank" stretch flat no-caps :label="version" class="btn-menu version" />
          </div>
          <q-tabs align="left" inline-label indicator-color="transparent" active-color="white" stretch>
            <q-route-tab to="/" icon="eva-home-outline" no-caps label="Dashboard" />
            <q-route-tab to="/http" icon="eva-globe-outline" no-caps label="HTTP" />
            <q-route-tab to="/tcp" icon="eva-globe-2-outline" no-caps label="TCP" />
            <q-route-tab to="/udp" icon="eva-globe-2-outline" no-caps label="UDP" />
          </q-tabs>
          <div class="right-menu">
            <q-tabs>
              <q-btn @click="$q.dark.toggle()" stretch flat no-caps icon="invert_colors" :label="`${$q.dark.isActive ? 'Light' : 'Dark'} theme`" class="btn-menu" />
              <q-btn stretch flat icon="eva-question-mark-circle-outline">
                <q-menu anchor="bottom left" auto-close>
                  <q-item>
                    <q-btn type="a" :href="`https://doc.traefik.io/traefik/${parsedVersion}`" target="_blank" flat color="accent" align="left" icon="eva-book-open-outline" no-caps label="Documentation" class="btn-submenu full-width"/>
                  </q-item>
                  <q-separator />
                  <q-item>
                    <q-btn type="a" href="https://github.com/traefik/traefik/" target="_blank" flat color="accent" align="left" icon="eva-github-outline" no-caps label="Github repository" class="btn-submenu full-width"/>
                  </q-item>
                </q-menu>
              </q-btn>
            </q-tabs>
            <platform-auth-state
              v-if="pilotEnabled" />
          </div>
        </q-toolbar>
      </div>
    </section>

    <section class="app-section text-black sub-nav" :class="{ 'bg-white': !$q.dark.isActive }">
      <div class="app-section-wrap app-boxed app-boxed-xl">
        <slot />
      </div>
    </section>
  </q-header>
</template>

<script>
import config from '../../../package'
import PlatformAuthState from '../platform/PlatformAuthState'
import { mapActions, mapGetters } from 'vuex'

export default {
  name: 'NavBar',
  components: { PlatformAuthState },
  computed: {
    ...mapGetters('core', { coreVersion: 'version' }),
    version () {
      if (!this.coreVersion.Version) return null
      return /^(v?\d+\.\d+)/.test(this.coreVersion.Version)
        ? this.coreVersion.Version
        : this.coreVersion.Version.substring(0, 7)
    },
    pilotEnabled () {
      return this.coreVersion.pilotEnabled
    },
    parsedVersion () {
      if (!this.version) {
        return 'master'
      }
      if (this.version === 'dev') {
        return 'master'
      }
      const matches = this.version.match(/^(v?\d+\.\d+)/)
      return matches ? 'v' + matches[1] : 'master'
    },
    name () {
      return config.productName
    }
  },
  methods: {
    ...mapActions('core', { getVersion: 'getVersion' })
  },
  created () {
    this.getVersion()
  }
}
</script>

<style scoped lang="scss">
  @import "../../css/sass/variables";

  .q-toolbar {
    min-height: 64px;
  }

  .body--dark {
    .sub-nav {
      background-color: #0e204c;
    }
  }

  .q-item--dark {
    background: var(--q-color-dark);
  }

  .logo {
    display: flex;
    align-items: center;

    img {
      height: 24px;
      margin-right: 10px;
    }

    .version {
      min-height: inherit;
      line-height:  inherit;
      padding: 0 4px;
    }
  }

  .q-tabs {
    color: rgba( $app-text-white, .4 );
    /deep/ .q-tabs__content {
      .q-tab__content{
        min-width: 100%;
        .q-tab__label {
          font-size: 16px;
          font-weight: 600;
        }
      }
    }
  }

  .right-menu {
    flex: 1;
    height: 64px;
    display: flex;
    justify-content: flex-end;
  }

  .btn-menu {
    color: rgba( $app-text-white, .4 );
    font-size: 16px;
    font-weight: 600;
  }

  .q-item {
    padding: 0;
  }

  .btn-submenu {
    font-weight: 700;
    align-items: flex-start;
  }
</style>
