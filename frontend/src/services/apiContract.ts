import type { mockApi } from './mock/serviceLayerMock';

export type ApiContract = typeof mockApi;
export type ServiceNamespace = keyof ApiContract;
export type ServiceMethodNames<TNamespace extends ServiceNamespace> = keyof ApiContract[TNamespace];
