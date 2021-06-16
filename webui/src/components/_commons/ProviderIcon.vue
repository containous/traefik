<template>
  <q-avatar class="provider-logo">
    <q-icon :name="`img:${getLogoPath}`" />
  </q-avatar>
</template>

<script>
export default {
  props: ['name'],
  computed: {
    getLogoPath () {
      const name = this.name.toLowerCase()

      if (name.includes('plugin-')) {
        return 'statics/providers/plugin.svg'
      }

      return `statics/providers/${name}.svg`
    }
  }
}
</script>

<style scoped lang="scss">
  .provider-logo {
    width: 32px;
    height: 32px;
    img {
      width: 100%;
      height: 100%;
    }
  }
</style>
