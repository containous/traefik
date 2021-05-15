<template>
  <q-card flat bordered>
    <q-card-section>
      <div class="row items-center no-wrap">
        <div class="col text-center">
          <q-avatar class="provider-logo">
            <q-icon :name="`img:${getLogoPath}`" />
          </q-avatar>
        </div>
      </div>
    </q-card-section>
    <q-card-section>
      <div class="text-h6 text-center text-weight-bold">
        {{getName}}
      </div>
    </q-card-section>
  </q-card>
</template>

<script>
export default {
  name: 'PanelProvider',
  props: ['name'],
  computed: {
    getName () {
      return this.name
    },
    getLogoPath () {
      const name = this.getName.toLowerCase()

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

  .provider-logo {
    width: 52px;
    height: 52px;
    img {
      width: 100%;
      height: 100%;
    }
  }
</style>
