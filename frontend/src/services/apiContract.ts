import type { realApi } from './serviceLayerReal';

export type ApiContract = typeof realApi;
export type ServiceNamespace = keyof ApiContract;
export type ServiceMethodNames<TNamespace extends ServiceNamespace> = keyof ApiContract[TNamespace];
