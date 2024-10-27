import type { Spec } from './v1alpha1/tdl';

export * as spec from './spec';
export * as v1alpha1 from './v1alpha1';

export type Gen = {
	// TODO
	(spec: Spec): Error
}
