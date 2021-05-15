<template>
  <q-card flat bordered v-bind:class="['panel-middleware-details', {'panel-middleware-details-dense':isDense}]">
    <q-scroll-area v-if="data && data.length" :thumb-style="appThumbStyle" style="height:100%;">
      <div v-for="(middleware, index) in data" :key="index">
        <q-card-section v-if="!isDense" class="app-title">
          <div class="app-title-label">{{ middleware.name }}</div>
        </q-card-section>
        <!-- COMMON FIELDS -->
        <q-card-section>
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Type</div>
              <q-chip
                dense
                class="app-chip app-chip-purple">
                {{ middleware.type }}
              </q-chip>
            </div>
            <div class="col">
              <div class="text-subtitle2">PROVIDER</div>
              <div class="block-right-text">
                <q-avatar class="provider-logo">
                  <q-icon :name="`img:${getProviderLogoPath(middleware.provider)}`" />
                </q-avatar>
                <div class="block-right-text-label">{{middleware.provider}}</div>
              </div>
            </div>
          </div>
        </q-card-section>
        <q-card-section>
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">STATUS</div>
              <div class="block-right-text">
                <avatar-state :state="middleware.status | status "/>
                <div v-bind:class="['block-right-text-label', `block-right-text-label-${middleware.status}`]">{{middleware.status | statusLabel}}</div>
              </div>
            </div>
          </div>
        </q-card-section>
        <!-- ERROR -->
        <q-card-section v-if="middleware.error">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">ERRORS</div>
              <q-chip
                v-for="(errorMsg, index) in middleware.error" :key="index"
                class="app-chip app-chip-error">
                {{ errorMsg }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [addPrefix] - prefix -->
        <q-card-section v-if="middleware.addPrefix">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">PREFIX</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).prefix }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [basicAuth & digestAuth] - users -->
        <q-card-section v-if="exData(middleware).users">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">USERS</div>
              <q-chip
                v-for="(user, key) in exData(middleware).users" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ user }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [basicAuth & digestAuth] - usersFile -->
        <q-card-section v-if="exData(middleware).usersFile">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Users File</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).usersFile }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [basicAuth & digestAuth] - realm -->
        <q-card-section v-if="exData(middleware).realm">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Realm</div>
              <q-chip
                dense
                class="app-chip app-chip-warning">
                {{ exData(middleware).realm }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [basicAuth & digestAuth] - removeHeader -->
        <q-card-section v-if="middleware.basicAuth || middleware.digestAuth">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Remove Header</div>
              <boolean-state :value="exData(middleware).removeHeader"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [basicAuth & digestAuth] - headerField -->
        <q-card-section v-if="exData(middleware).headerField">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Header Field</div>
              <q-chip
                dense
                class="app-chip app-chip-warning">
                {{ exData(middleware).headerField }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [chain] - middlewares -->
        <q-card-section v-if="middleware.chain">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Chain</div>
              <q-chip
                v-for="(mi, key) in exData(middleware).middlewares" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ mi }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [buffering] - xxxRequestBodyBytes -->
        <q-card-section v-if="exData(middleware).maxRequestBodyBytes || exData(middleware).memRequestBodyBytes">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Max Request Body Bytes</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).maxRequestBodyBytes }}
              </q-chip>
            </div>
            <div class="col">
              <div class="text-subtitle2">Mem Request Body Bytes</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).memRequestBodyBytes }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [buffering] - xxxResponseBodyBytes -->
        <q-card-section v-if="exData(middleware).maxResponseBodyBytes || exData(middleware).memResponseBodyBytes">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Max Response Body Bytes</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).maxResponseBodyBytes }}
              </q-chip>
            </div>
            <div class="col">
              <div class="text-subtitle2">Mem Response Body Bytes</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).memResponseBodyBytes }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [buffering] - retryExpression -->
        <q-card-section v-if="exData(middleware).retryExpression">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Retry Expression</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).retryExpression }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [circuitBreaker] - expression -->
        <q-card-section v-if="middleware.circuitBreaker">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Expression</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).expression }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [compress] - compress -->
        <q-card-section v-if="middleware.compress">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Compress</div>
              <boolean-state :value="!!middleware.compress"/>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [errors] - service -->
        <q-card-section v-if="middleware.errors">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Service</div>
              <q-chip
                dense
                class="app-chip app-chip-green app-chip-overflow">
                {{ exData(middleware).service }}
                <q-tooltip>{{ exData(middleware).service }}</q-tooltip>
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [errors] - query -->
        <q-card-section v-if="middleware.errors">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Query</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).query }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [errors] - status -->
        <q-card-section v-if="middleware.errors">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Status</div>
              <q-chip
                v-for="(st, key) in exData(middleware).status" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ st }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [forwardAuth] - address -->
        <q-card-section v-if="middleware.forwardAuth">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Address</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).address }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [forwardAuth] - trustForwardHeader && tls -->
        <q-card-section v-if="middleware.forwardAuth">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">TLS</div>
              <boolean-state :value="!!exData(middleware).tls"/>
            </div>
            <div class="col">
              <div class="text-subtitle2">Trust Forward Headers</div>
              <boolean-state :value="exData(middleware).trustForwardHeader"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [forwardAuth] - authResponseHeaders -->
        <q-card-section v-if="middleware.forwardAuth">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Auth Response Headers</div>
              <q-chip
                v-for="(respHeader, key) in exData(middleware).authResponseHeaders" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ respHeader }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [forwardAuth] - authRequestHeaders -->
        <q-card-section v-if="middleware.forwardAuth">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Auth Request Headers</div>
              <q-chip
                v-for="(reqHeader, key) in exData(middleware).authRequestHeaders" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ reqHeader }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - customRequestHeaders -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Custom Request Headers</div>
              <q-chip
                v-for="(val, key) in exData(middleware).customRequestHeaders" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ key }}: {{ val }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - customResponseHeaders -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Custom Response Headers</div>
              <q-chip
                v-for="(val, key) in exData(middleware).customResponseHeaders" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ key }}: {{ val }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - accessControlAllowCredentials -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Access Control Allow Credentials</div>
              <boolean-state :value="!!exData(middleware).accessControlAllowCredentials"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - accessControlAllowHeaders -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Access Control Allow Headers</div>
              <q-chip
                v-for="(val, key) in exData(middleware).accessControlAllowHeaders" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ val }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - accessControlAllowMethods -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Access Control Allow Methods</div>
              <q-chip
                v-for="(val, key) in exData(middleware).accessControlAllowMethods" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ val }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - accessControlAllowOriginList -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Access Control Allow Origin</div>
              <q-chip
                v-for="(val, key) in exData(middleware).accessControlAllowOriginList" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ val }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - accessControlExposeHeaders -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Access Control Expose Headers</div>
              <q-chip
                v-for="(val, key) in exData(middleware).accessControlExposeHeaders" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ val }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - accessControlMaxAge -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Access Control Max Age</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).accessControlMaxAge }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - addVaryHeader -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Add Vary Header</div>
              <boolean-state :value="!!exData(middleware).addVaryHeader"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - allowedHosts -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Allowed Hosts</div>
              <q-chip
                v-for="(val, key) in exData(middleware).allowedHosts" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ val }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - hostsProxyHeaders -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Hosts Proxy Headers</div>
              <q-chip
                v-for="(val, key) in exData(middleware).hostsProxyHeaders" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ val }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - sslRedirect -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">SSL Redirect</div>
              <boolean-state :value="!!exData(middleware).sslRedirect"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - sslTemporaryRedirect -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">SSL Temporary Redirect</div>
              <boolean-state :value="!!exData(middleware).sslTemporaryRedirect"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - sslHost -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">SSL Host</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).sslHost }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - sslProxyHeaders -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">SSL Proxy Headers</div>
              <q-chip
                v-for="(val, key) in exData(middleware).sslProxyHeaders" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ val }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - sslForceHost -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">SSL Force Host</div>
              <boolean-state :value="!!exData(middleware).sslForceHost"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - stsSeconds -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">STS Seconds</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).stsSeconds }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - stsIncludeSubdomains -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">STS Include Subdomains</div>
              <boolean-state :value="!!exData(middleware).stsIncludeSubdomains"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - stsPreload -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">STS Preload</div>
              <boolean-state :value="!!exData(middleware).stsPreload"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - forceSTSHeader -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Force STS Header</div>
              <boolean-state :value="!!exData(middleware).forceSTSHeader"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - frameDeny -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Frame Deny</div>
              <boolean-state :value="!!exData(middleware).frameDeny"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - customFrameOptionsValue -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Custom Frame Options Value</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).customFrameOptionsValue }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - contentTypeNosniff -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Content Type Nosniff</div>
              <boolean-state :value="!!exData(middleware).contentTypeNosniff"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - browserXssFilter -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Browser XSS Filter</div>
              <boolean-state :value="!!exData(middleware).browserXssFilter"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - customBrowserXSSValue -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Custom Browser XSS Value</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).customBrowserXSSValue }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - contentSecurityPolicy -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Content Security Policy</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).contentSecurityPolicy }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - publicKey -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Public Key</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).publicKey }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - referrerPolicy -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Referrer Policy</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).referrerPolicy }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - featurePolicy -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Feature Policy</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).featurePolicy }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [headers] - isDevelopment -->
        <q-card-section v-if="middleware.headers">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Is Development</div>
              <boolean-state :value="!!exData(middleware).isDevelopment"/>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [ipWhiteList] - sourceRange -->
        <q-card-section v-if="middleware.ipWhiteList">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Source Range</div>
              <q-chip
                v-for="(range, key) in exData(middleware).sourceRange" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ range }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [ipWhiteList] - ipStrategy -->
        <q-card-section v-if="middleware.ipWhiteList">
          <div class="row items-start">
            <div class="col-12">
              <div class="text-subtitle2">IP Strategy</div>
            </div>
            <div v-if="exData(middleware).ipStrategy && exData(middleware).ipStrategy.depth" class="col-12">
              <q-chip
                dense
                class="app-chip app-chip-accent">Depth :</q-chip>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).ipStrategy.depth }}
              </q-chip>
            </div>
            <div v-if="exData(middleware).ipStrategy && exData(middleware).ipStrategy.excludedIPs" class="col-12">
              <div class="flex">
                <q-chip
                  dense
                  class="app-chip app-chip-accent">
                  Excluded IPs:
                </q-chip>
                <q-chip
                  v-for="(excludedIPs, key) in exData(middleware).ipStrategy.excludedIPs" :key="key"
                  dense
                  class="app-chip app-chip-green">
                  {{ excludedIPs }}
                </q-chip>
              </div>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [rateLimit] - average && burst-->
        <q-card-section v-if="middleware.rateLimit">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Average</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).average }}
              </q-chip>
            </div>
            <div class="col">
              <div class="text-subtitle2">Burst</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).burst }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [inFlightReq] - amount -->
        <q-card-section v-if="exData(middleware).amount">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">AMOUNT</div>
              <q-chip
                dense
                class="app-chip app-chip-warning">
                {{ exData(middleware).amount }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [inFlightReq & rateLimit] - ipStrategy -->
        <q-card-section v-if="exData(middleware).sourceCriterion && exData(middleware).sourceCriterion.ipStrategy">
          <div class="row items-start">
            <div class="col-12">
              <div class="text-subtitle2">IP STRATEGY</div>
            </div>
            <div v-if="exData(middleware).sourceCriterion.ipStrategy.depth" class="col-12">
              <q-chip
                dense
                class="app-chip app-chip-accent">Depth :</q-chip>
              <q-chip
                dense
                class="app-chip app-chip-green">
               {{ exData(middleware).sourceCriterion.ipStrategy.depth }}
              </q-chip>
            </div>
            <div v-if="exData(middleware).sourceCriterion.ipStrategy.excludedIPs" class="col-12">
              <div class="flex">
                <q-chip
                  dense
                  class="app-chip app-chip-accent">
                  Excluded IPs:
                </q-chip>
                <q-chip
                  v-for="(excludedIPs, key) in exData(middleware).sourceCriterion.ipStrategy.excludedIPs" :key="key"
                  dense
                  class="app-chip app-chip-green">
                  {{ excludedIPs }}
                </q-chip>
              </div>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [inFlightReq & rateLimit] - requestHeaderName, requestHost -->
        <q-card-section v-if="exData(middleware) && exData(middleware).sourceCriterion">
          <div class="row items-start no-wrap">
            <div v-if="exData(middleware).sourceCriterion.requestHeaderName" class="col">
              <div class="text-subtitle2">REQUEST HEADER NAME</div>
              <q-chip
                dense
                class="app-chip app-chip-warning">
                {{ exData(middleware).sourceCriterion.requestHeaderName }}
              </q-chip>
            </div>
            <div v-if="exData(middleware).sourceCriterion.requestHost" class="col">
              <div class="text-subtitle2">REQUEST HOST</div>
              <boolean-state :value="exData(middleware).sourceCriterion.requestHost"/>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [passTLSClientCert] - pem -->
        <q-card-section v-if="middleware.passTLSClientCert">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">PEM</div>
              <boolean-state :value="!!exData(middleware).pem"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [passTLSClientCert] - info - notAfter -->
        <q-card-section v-if="middleware.passTLSClientCert && middleware.passTLSClientCert.info">
          <div class="text-subtitle2">Info:</div>
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Not After</div>
              <boolean-state :value="!!exData(middleware).info.notAfter"/>
            </div>
            <div class="col">
              <div class="text-subtitle2">Not Before</div>
              <boolean-state :value="!!exData(middleware).info.notBefore"/>
            </div>
            <div class="col">
              <div class="text-subtitle2">Sans</div>
              <boolean-state :value="!!exData(middleware).info.sans"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [passTLSClientCert] - info - subject -->
        <q-card-section v-if="middleware.passTLSClientCert && middleware.passTLSClientCert.info && middleware.passTLSClientCert.info.subject">
          <div class="text-subtitle2">Info Subject:</div>
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">country</div>
              <boolean-state :value="!!exData(middleware).info.subject.country"/>
            </div>
            <div class="col">
              <div class="text-subtitle2">Province</div>
              <boolean-state :value="!!exData(middleware).info.subject.province"/>
            </div>
          </div>
        </q-card-section>
        <q-card-section v-if="middleware.passTLSClientCert && middleware.passTLSClientCert.info && middleware.passTLSClientCert.info.subject">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Locality</div>
              <boolean-state :value="!!exData(middleware).info.subject.locality"/>
            </div>
            <div class="col">
              <div class="text-subtitle2">Organization</div>
              <boolean-state :value="!!exData(middleware).info.subject.organization"/>
            </div>
          </div>
        </q-card-section>
        <q-card-section v-if="middleware.passTLSClientCert && middleware.passTLSClientCert.info && middleware.passTLSClientCert.info.subject">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Common Name</div>
              <boolean-state :value="!!exData(middleware).info.subject.commonName"/>
            </div>
            <div class="col">
              <div class="text-subtitle2">Serial Number</div>
              <boolean-state :value="!!exData(middleware).info.subject.serialNumber"/>
            </div>
          </div>
        </q-card-section>
        <q-card-section v-if="middleware.passTLSClientCert && middleware.passTLSClientCert.info && middleware.passTLSClientCert.info.subject">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Domain Component</div>
              <boolean-state :value="!!exData(middleware).info.subject.domainComponent"/>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [passTLSClientCert] - info - issuer -->
        <q-card-section v-if="middleware.passTLSClientCert && middleware.passTLSClientCert.info && middleware.passTLSClientCert.info.issuer">
          <div class="text-subtitle2">Info Issuer:</div>
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">country</div>
              <boolean-state :value="!!exData(middleware).info.issuer.country"/>
            </div>
            <div class="col">
              <div class="text-subtitle2">Province</div>
              <boolean-state :value="!!exData(middleware).info.issuer.province"/>
            </div>
          </div>
        </q-card-section>
        <q-card-section v-if="middleware.passTLSClientCert && middleware.passTLSClientCert.info && middleware.passTLSClientCert.info.issuer">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Locality</div>
              <boolean-state :value="!!exData(middleware).info.issuer.locality"/>
            </div>
            <div class="col">
              <div class="text-subtitle2">Organization</div>
              <boolean-state :value="!!exData(middleware).info.issuer.organization"/>
            </div>
          </div>
        </q-card-section>
        <q-card-section v-if="middleware.passTLSClientCert && middleware.passTLSClientCert.info && middleware.passTLSClientCert.info.issuer">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Common Name</div>
              <boolean-state :value="!!exData(middleware).info.issuer.commonName"/>
            </div>
            <div class="col">
              <div class="text-subtitle2">Serial Number</div>
              <boolean-state :value="!!exData(middleware).info.issuer.serialNumber"/>
            </div>
          </div>
        </q-card-section>
        <q-card-section v-if="middleware.passTLSClientCert && middleware.passTLSClientCert.info && middleware.passTLSClientCert.info.issuer">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Domain Component</div>
              <boolean-state :value="!!exData(middleware).info.issuer.domainComponent"/>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [redirectRegex] - regex -->
        <q-card-section v-if="middleware.redirectRegex">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Regex</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).regex }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [redirectRegex] - replacement -->
        <q-card-section v-if="middleware.redirectRegex">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Replacement</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).replacement }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [redirectRegex] - permanent -->
        <q-card-section v-if="middleware.redirectRegex">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Permanent</div>
              <boolean-state :value="!!exData(middleware).permanent"/>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [redirectScheme] - scheme -->
        <q-card-section v-if="middleware.redirectScheme">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Scheme</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).scheme }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [replacePath] - path -->
        <q-card-section v-if="middleware.replacePath">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Path</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).path }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [replacePathRegex] - regex -->
        <q-card-section v-if="middleware.replacePathRegex">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Regex</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).regex }}
              </q-chip>
            </div>
          </div>
        </q-card-section>
        <!-- EXTRA FIELDS FROM MIDDLEWARES - [replacePathRegex] - replacement -->
        <q-card-section v-if="middleware.replacePathRegex">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Replacement</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).replacement }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [retry] - attempts -->
        <q-card-section v-if="middleware.retry">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Attempts</div>
              <q-chip
                dense
                class="app-chip app-chip-green">
                {{ exData(middleware).attempts }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [stripPrefix] - prefixes -->
        <q-card-section v-if="middleware.stripPrefix">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Prefixes</div>
              <q-chip
                v-for="(prefix, key) in exData(middleware).prefixes" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ prefix }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <!-- EXTRA FIELDS FROM MIDDLEWARES - [stripPrefixRegex] - regex -->
        <q-card-section v-if="middleware.stripPrefixRegex">
          <div class="row items-start no-wrap">
            <div class="col">
              <div class="text-subtitle2">Regex</div>
              <q-chip
                v-for="(exp, key) in exData(middleware).regex" :key="key"
                dense
                class="app-chip app-chip-green">
                {{ exp }}
              </q-chip>
            </div>
          </div>
        </q-card-section>

        <q-separator v-if="(index+1) < data.length" inset />
      </div>
    </q-scroll-area>
    <q-card-section v-else style="height: 100%">
      <div class="row items-center" style="height: 100%">
        <div class="col-12">
        <div class="block-empty"></div>
          <div class="q-pb-lg block-empty-logo">
            <img v-if="$q.dark.isActive" alt="empty" src="~assets/middlewares-empty-dark.svg">
            <img v-else alt="empty" src="~assets/middlewares-empty.svg">
          </div>
          <div class="block-empty-label">There are no<br>Middlewares configured</div>
        </div>
      </div>
    </q-card-section>
  </q-card>
</template>

<script>
import BooleanState from './BooleanState'
import AvatarState from './AvatarState'

export default {
  name: 'PanelMiddlewareDetails',
  props: ['data', 'dense'],
  components: {
    AvatarState,
    BooleanState
  },
  computed: {
    isDense () {
      return this.dense !== undefined
    }
  },
  methods: {
    exData (item) {
      let exData = {}
      for (const prop in item) {
        if (prop.toLowerCase() === item.type && item.hasOwnProperty(prop)) {
          exData = item[prop]
        }
      }
      return exData
    },
    getProviderLogoPath (provider) {
      const name = provider.name.toLowerCase()

      if (name.includes('plugin-')) {
        return 'statics/providers/plugin.svg'
      }

      return `statics/providers/${name}.svg`
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
  }
}
</script>

<style scoped lang="scss">
  @import "../../css/sass/variables";

  .panel-middleware-details {
    height: 600px;
    &-dense{
      /*height: 400px;*/
    }
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
