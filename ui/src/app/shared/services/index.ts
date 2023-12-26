// This is the main entry point for the services used in the application
import { HelmCatalogService } from './helmchart-catalog.service';
import { HelmChartReleaseService } from './helmchart-release.service';
import { K3sService } from './k3s.service';
import { WebsocketService } from './websocket.service';
/**
 * HelmCatalogService is responsible for pulling in information from chart repositories.
 * This includes chart manifests, versions, readmes, and other related data. It does not interact with the Kubernetes API directly.
 */
/**
 * HelmChartReleaseService is used for installing, upgrading and uninstalling Helm chart releases.
 * It also pulls information about releases from the Kubernetes API, managing the lifecycle of Helm charts within a cluster.
 */
/**
 * K3sService is responsible for installing, upgrading and uninstalling K3s clusters.
 */

export { HelmCatalogService, HelmChartReleaseService, K3sService, WebsocketService }
