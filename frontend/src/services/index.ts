import { config } from '@/config';
import type { ApiContract } from './apiContract';
import { mockApi, resetDemoData, MODULE_LABELS, STATUS_ORDER_EXPORT } from './mock/serviceLayerMock';
import { realApi } from './serviceLayerReal';

export const api: ApiContract = (config.apiBaseUrl ? realApi : mockApi) as ApiContract;
export type Api = ApiContract;
export { resetDemoData, MODULE_LABELS, STATUS_ORDER_EXPORT };
