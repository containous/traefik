import { withPagination } from '../../_helpers/Mutations'

// ----------------------------
// Get All Routers
// ----------------------------
export function getAllRoutersRequest (state) {
  withPagination('request', { statePath: 'allRouters' })(state)
}

export function getAllRoutersSuccess (state, data) {
  const { query = '', status = '' } = data
  const currentState = state.allRouters

  const isSameContext = currentState.currentQuery === query && currentState.currentStatus === status

  state.allRouters = {
    ...state.allRouters,
    currentQuery: query,
    currentStatus: status
  }

  withPagination('success', {
    isSameContext,
    statePath: 'allRouters'
  })(state, data)
}

export function getAllRoutersFailure (state, error) {
  withPagination('failure', { statePath: 'allRouters' })(state, error)
}

export function getAllRoutersClear (state) {
  state.allRouters = {}
}

// ----------------------------
// Get Router By Name
// ----------------------------
export function getRouterByNameRequest (state) {
  state.routerByName.loading = true
}

export function getRouterByNameSuccess (state, body) {
  state.routerByName = { item: body, loading: false }
}

export function getRouterByNameFailure (state, error) {
  state.routerByName = { error }
}

export function getRouterByNameClear (state) {
  state.routerByName = {}
}

// ----------------------------
// Get All Services
// ----------------------------
export function getAllServicesRequest (state) {
  withPagination('request', { statePath: 'allServices' })(state)
}

export function getAllServicesSuccess (state, data) {
  const { query = '', status = '' } = data
  const currentState = state.allServices

  const isSameContext = currentState.currentQuery === query && currentState.currentStatus === status

  state.allServices = {
    ...state.allServices,
    currentQuery: query,
    currentStatus: status
  }

  withPagination('success', {
    isSameContext,
    statePath: 'allServices'
  })(state, data)
}

export function getAllServicesFailure (state, error) {
  withPagination('failure', { statePath: 'allServices' })(state, error)
}

export function getAllServicesClear (state) {
  state.allServices = {}
}

// ----------------------------
// Get Service By Name
// ----------------------------
export function getServiceByNameRequest (state) {
  state.serviceByName.loading = true
}

export function getServiceByNameSuccess (state, body) {
  state.serviceByName = { item: body, loading: false }
}

export function getServiceByNameFailure (state, error) {
  state.serviceByName = { error }
}

export function getServiceByNameClear (state) {
  state.serviceByName = {}
}

// ----------------------------
// Get All Middlewares
// ----------------------------
export function getAllMiddlewaresRequest (state) {
  withPagination('request', { statePath: 'allMiddlewares' })(state)
}

export function getAllMiddlewaresSuccess (state, data) {
  const { query = '', status = '' } = data
  const currentState = state.allMiddlewares

  const isSameContext = currentState.currentQuery === query && currentState.currentStatus === status

  state.allMiddlewares = {
    ...state.allMiddlewares,
    currentQuery: query,
    currentStatus: status
  }

  withPagination('success', {
    isSameContext,
    statePath: 'allMiddlewares'
  })(state, data)
}

export function getAllMiddlewaresFailure (state, error) {
  withPagination('failure', { statePath: 'allMiddlewares' })(state, error)
}

export function getAllMiddlewaresClear (state) {
  state.allMiddlewares = {}
}

// ----------------------------
// Get Middleware By Name
// ----------------------------
export function getMiddlewareByNameRequest (state) {
  state.middlewareByName.loading = true
}

export function getMiddlewareByNameSuccess (state, body) {
  state.middlewareByName = { item: body, loading: false }
}

export function getMiddlewareByNameFailure (state, error) {
  state.middlewareByName = { error }
}

export function getMiddlewareByNameClear (state) {
  state.middlewareByName = {}
}
