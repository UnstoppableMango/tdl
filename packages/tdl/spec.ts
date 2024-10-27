import * as buf from '@bufbuild/protobuf';
import type { MessageInitShape } from '@bufbuild/protobuf';
import type { GenMessage } from '@bufbuild/protobuf/codegenv1';
import { type Spec, SpecSchema } from './v1alpha1/tdl';

export function create(init?: MessageInitShape<GenMessage<Spec>>): Spec {
	return buf.create(SpecSchema, init);
}
