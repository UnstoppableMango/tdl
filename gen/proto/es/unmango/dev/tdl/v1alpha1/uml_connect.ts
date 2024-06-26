// @generated by protoc-gen-connect-es v1.4.0 with parameter "target=ts,import_extension=none"
// @generated from file unmango/dev/tdl/v1alpha1/uml.proto (package unmango.dev.tdl.v1alpha1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import { FromRequest, FromResponse, GenRequest, GenResponse } from "./uml_pb";
import { MethodKind } from "@bufbuild/protobuf";

/**
 * @generated from service unmango.dev.tdl.v1alpha1.UmlService
 */
export const UmlService = {
  typeName: "unmango.dev.tdl.v1alpha1.UmlService",
  methods: {
    /**
     * @generated from rpc unmango.dev.tdl.v1alpha1.UmlService.From
     */
    from: {
      name: "From",
      I: FromRequest,
      O: FromResponse,
      kind: MethodKind.ClientStreaming,
    },
    /**
     * @generated from rpc unmango.dev.tdl.v1alpha1.UmlService.Gen
     */
    gen: {
      name: "Gen",
      I: GenRequest,
      O: GenResponse,
      kind: MethodKind.ServerStreaming,
    },
  }
} as const;

