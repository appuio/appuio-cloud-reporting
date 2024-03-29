local c = import 'common.libsonnet';

local query = importstr '../appuio_managed_openshift_vcpu.promql';

local commonLabels = {
  cluster_id: 'c-managed-openshift',
};

local infoLabels = commonLabels {
  tenant_id: 't-managed-openshift',
  vshn_service_level: 'ondemand',
  cloud_provider: 'cloudscale',
};

local baseSeries = {
  appNodeRoleLabel: c.series('kube_node_role', commonLabels {
    node: 'app-test',
    role: 'app',
  }, '1x120'),

  appNodeCPUInfoLabel0: c.series('node_cpu_info', commonLabels {
    instance: 'app-test',
    core: '0',
  }, '1x120'),
  appNodeCPUInfoLabel1: c.series('node_cpu_info', commonLabels {
    instance: 'app-test',
    core: '1',
  }, '1x120'),

  storageNodeRoleLabel: c.series('kube_node_role', commonLabels {
    node: 'storage-test',
    role: 'storage',
  }, '1x120'),

  storageNodeCPUInfoLabel0: c.series('node_cpu_info', commonLabels {
    instance: 'storage-test',
    core: '0',
  }, '1x120'),

  appuioInfoLabel: c.series('appuio_managed_info', infoLabels, '1x120'),
};

local baseCalculatedLabels = infoLabels {
  class: super.vshn_service_level,
  category: super.tenant_id + ':' + super.cluster_id,
};

{
  tests: [
    c.test(
      'two app CPUs and one storage CPU',
      baseSeries,
      query,
      [
        {
          labels: c.formatLabels(baseCalculatedLabels {
            role: 'app',
            product: 'appuio_managed_openshift_vcpu:cloudscale:t-managed-openshift:c-managed-openshift:ondemand:app',
          }),
          value: 2,
        },
        {
          labels: c.formatLabels(baseCalculatedLabels {
            role: 'storage',
            product: 'appuio_managed_openshift_vcpu:cloudscale:t-managed-openshift:c-managed-openshift:ondemand:storage',
          }),
          value: 1,
        },
      ]
    ),

  ],
}
