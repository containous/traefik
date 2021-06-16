const LayoutDefault = () => import('layouts/Default.vue')

const routes = [
  {
    path: '/',
    component: LayoutDefault,
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('pages/dashboard/Index.vue'),
        meta: {
          title: 'Dashboard'
        }
      }
    ]
  },
  {
    path: '/http',
    redirect: '/http/routers',
    component: LayoutDefault,
    children: [
      {
        path: 'routers',
        name: 'httpRouters',
        components: {
          default: () => import('pages/http/Routers.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'http',
          title: 'HTTP Routers'
        }
      },
      {
        path: 'routers/:name',
        name: 'httpRouterDetail',
        components: {
          default: () => import('pages/_commons/RouterDetail.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'http',
          title: 'HTTP Router Detail'
        }
      },
      {
        path: 'services',
        name: 'httpServices',
        components: {
          default: () => import('pages/http/Services.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'http',
          title: 'HTTP Services'
        }
      },
      {
        path: 'services/:name',
        name: 'httpServiceDetail',
        components: {
          default: () => import('pages/_commons/ServiceDetail.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'http',
          title: 'HTTP Service Detail'
        }
      },
      {
        path: 'middlewares',
        name: 'httpMiddlewares',
        components: {
          default: () => import('pages/http/Middlewares.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'http',
          title: 'HTTP Middlewares'
        }
      },
      {
        path: 'middlewares/:name',
        name: 'httpMiddlewareDetail',
        components: {
          default: () => import('pages/_commons/MiddlewareDetail.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'http',
          title: 'HTTP Middleware Detail'
        }
      }
    ]
  },
  {
    path: '/tcp',
    redirect: '/tcp/routers',
    component: LayoutDefault,
    children: [
      {
        path: 'routers',
        name: 'tcpRouters',
        components: {
          default: () => import('pages/tcp/Routers.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'tcp',
          title: 'TCP Routers'
        }
      },
      {
        path: 'routers/:name',
        name: 'tcpRouterDetail',
        components: {
          default: () => import('pages/_commons/RouterDetail.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'tcp',
          title: 'TCP Router Detail'
        }
      },
      {
        path: 'services',
        name: 'tcpServices',
        components: {
          default: () => import('pages/tcp/Services.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'tcp',
          title: 'TCP Services'
        }
      },
      {
        path: 'services/:name',
        name: 'tcpServiceDetail',
        components: {
          default: () => import('pages/_commons/ServiceDetail.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'tcp',
          title: 'TCP Service Detail'
        }
      },
      {
        path: 'middlewares',
        name: 'tcpMiddlewares',
        components: {
          default: () => import('pages/tcp/Middlewares.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'tcp',
          title: 'TCP Middlewares'
        }
      },
      {
        path: 'middlewares/:name',
        name: 'tcpMiddlewareDetail',
        components: {
          default: () => import('pages/_commons/MiddlewareDetail.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'tcp',
          title: 'TCP Middleware Detail'
        }
      }
    ]
  },
  {
    path: '/udp',
    redirect: '/udp/routers',
    component: LayoutDefault,
    children: [
      {
        path: 'routers',
        name: 'udpRouters',
        components: {
          default: () => import('pages/udp/Routers.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'udp',
          title: 'TCP Routers'
        }
      },
      {
        path: 'routers/:name',
        name: 'udpRouterDetail',
        components: {
          default: () => import('pages/_commons/RouterDetail.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'udp',
          title: 'TCP Router Detail'
        }
      },
      {
        path: 'services',
        name: 'udpServices',
        components: {
          default: () => import('pages/udp/Services.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'udp',
          title: 'TCP Services'
        }
      },
      {
        path: 'services/:name',
        name: 'udpServiceDetail',
        components: {
          default: () => import('pages/_commons/ServiceDetail.vue'),
          NavBar: () => import('components/_commons/ToolBar.vue')
        },
        props: { default: true, NavBar: true },
        meta: {
          protocol: 'udp',
          title: 'TCP Service Detail'
        }
      }
    ]
  }
]

// Always leave this as last one
if (process.env.MODE !== 'ssr') {
  routes.push({
    path: '*',
    component: () => import('pages/_commons/Error404.vue'),
    meta: {
      title: '404'
    }
  })
}

export default routes
