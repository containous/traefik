import { get } from 'dot-prop'
import { QChip } from 'quasar'
import Chips from '../components/_commons/Chips'
import ProviderIcon from '../components/_commons/ProviderIcon'
import AvatarState from '../components/_commons/AvatarState'
import TLSState from '../components/_commons/TLSState'

const allColumns = [
  {
    name: 'status',
    required: true,
    label: 'Status',
    align: 'left',
    fieldToProps: row => ({
      state: row.status === 'enabled' ? 'positive' : 'negative'
    }),
    component: AvatarState
  },
  {
    name: 'tls',
    align: 'left',
    label: 'TLS',
    fieldToProps: row => ({ isTLS: row.tls }),
    component: TLSState
  },
  {
    name: 'rule',
    align: 'left',
    label: 'Rule',
    component: QChip,
    fieldToProps: () => ({ class: 'app-chip app-chip-rule', dense: true }),
    content: row => row.rule
  },
  {
    name: 'entryPoints',
    align: 'left',
    label: 'Entrypoints',
    component: Chips,
    fieldToProps: row => ({
      classNames: 'app-chip app-chip-entry-points',
      dense: true,
      list: row.entryPoints
    })
  },
  {
    name: 'name',
    align: 'left',
    label: 'Name',
    component: QChip,
    fieldToProps: () => ({ class: 'app-chip app-chip-name', dense: true }),
    content: row => row.name
  },
  {
    name: 'type',
    align: 'left',
    label: 'Type',
    component: QChip,
    fieldToProps: () => ({
      class: 'app-chip app-chip-entry-points',
      dense: true
    }),
    content: row => row.type
  },
  {
    name: 'servers',
    align: 'right',
    label: 'Servers',
    fieldToProps: () => ({ class: 'servers-label' }),
    content: function (value) {
      if (value.loadBalancer && value.loadBalancer.servers) {
        return value.loadBalancer.servers.length
      }
      return 0
    }
  },
  {
    name: 'service',
    align: 'left',
    label: 'Service',
    component: QChip,
    fieldToProps: () => ({ class: 'app-chip app-chip-service', dense: true }),
    content: row => row.service
  },
  {
    name: 'provider',
    align: 'center',
    label: 'Provider',
    fieldToProps: row => ({ name: row.provider }),
    component: ProviderIcon
  }
]

const columnsByResource = {
  routers: [
    'status',
    'rule',
    'entryPoints',
    'name',
    'service',
    'tls',
    'provider'
  ],
  udpRouters: ['status', 'entryPoints', 'name', 'service', 'provider'],
  services: ['status', 'name', 'type', 'servers', 'provider'],
  middlewares: ['status', 'name', 'type', 'provider']
}

const propsByType = {
  'http-routers': {
    columns: columnsByResource.routers
  },
  'tcp-routers': {
    columns: columnsByResource.routers
  },
  'udp-routers': {
    columns: columnsByResource.udpRouters
  },
  'http-services': {
    columns: columnsByResource.services
  },
  'tcp-services': {
    columns: columnsByResource.services
  },
  'udp-services': {
    columns: columnsByResource.services
  },
  'http-middlewares': {
    columns: columnsByResource.middlewares
  },
  'tcp-middlewares': {
    columns: columnsByResource.middlewares
  }
}

const GetTablePropsMixin = {
  methods: {
    getTableProps ({ type }) {
      return {
        onRowClick: row =>
          this.$router.push({
            path: `/${type.replace('-', '/', 'gi')}/${row.name}`
          }),
        columns: allColumns.filter(c =>
          get(propsByType, `${type}.columns`, []).includes(c.name)
        )
      }
    }
  }
}

export default GetTablePropsMixin
