/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Observable } from "rxjs";
import { map } from "rxjs/operators";
import { Any } from "../../../../google/protobuf/any";

export const protobufPackage = "unmango.dev.tdl.v1alpha1";

export interface FromRequest {
  data: Uint8Array;
}

export interface FromResponse {
  spec: Spec | undefined;
}

export interface GenRequest {
  spec: Spec | undefined;
}

export interface GenResponse {
  data: Uint8Array;
}

export interface ToRequest {
  spec: Spec | undefined;
}

export interface ToResponse {
  data: Uint8Array;
}

export interface Spec {
  name: string;
  source: string;
  version: string;
  displayName: string;
  description: string;
  labels: { [key: string]: string };
  types: { [key: string]: Type };
  functions: { [key: string]: Function };
  meta: { [key: string]: Any };
}

export interface Spec_LabelsEntry {
  key: string;
  value: string;
}

export interface Spec_TypesEntry {
  key: string;
  value: Type | undefined;
}

export interface Spec_FunctionsEntry {
  key: string;
  value: Function | undefined;
}

export interface Spec_MetaEntry {
  key: string;
  value: Any | undefined;
}

export interface Type {
  type: string;
  fields: { [key: string]: Field };
  methods: { [key: string]: Function };
  genericParameters: { [key: string]: GenericParameter };
  constructor?: Constructor | undefined;
  meta: { [key: string]: Any };
}

export interface Type_FieldsEntry {
  key: string;
  value: Field | undefined;
}

export interface Type_MethodsEntry {
  key: string;
  value: Function | undefined;
}

export interface Type_GenericParametersEntry {
  key: string;
  value: GenericParameter | undefined;
}

export interface Type_MetaEntry {
  key: string;
  value: Any | undefined;
}

export interface Field {
  type: string;
  readonly: boolean;
  meta: { [key: string]: Any };
}

export interface Field_MetaEntry {
  key: string;
  value: Any | undefined;
}

export interface Function {
  returnType: Type | undefined;
  parameters: { [key: string]: Type };
  genericParameters: { [key: string]: GenericParameter };
  meta: { [key: string]: Any };
}

export interface Function_ParametersEntry {
  key: string;
  value: Type | undefined;
}

export interface Function_GenericParametersEntry {
  key: string;
  value: GenericParameter | undefined;
}

export interface Function_MetaEntry {
  key: string;
  value: Any | undefined;
}

export interface GenericParameter {
  modifiers: Modifier[];
  meta: { [key: string]: Any };
}

export interface GenericParameter_MetaEntry {
  key: string;
  value: Any | undefined;
}

export interface Modifier {
  meta: { [key: string]: Any };
}

export interface Modifier_MetaEntry {
  key: string;
  value: Any | undefined;
}

export interface Constructor {
  parameters: { [key: string]: Type };
  meta: { [key: string]: Any };
}

export interface Constructor_ParametersEntry {
  key: string;
  value: Type | undefined;
}

export interface Constructor_MetaEntry {
  key: string;
  value: Any | undefined;
}

function createBaseFromRequest(): FromRequest {
  return { data: new Uint8Array(0) };
}

export const FromRequest = {
  encode(message: FromRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.data.length !== 0) {
      writer.uint32(10).bytes(message.data);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FromRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFromRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.data = reader.bytes();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FromRequest {
    return { data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(0) };
  },

  toJSON(message: FromRequest): unknown {
    const obj: any = {};
    if (message.data.length !== 0) {
      obj.data = base64FromBytes(message.data);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<FromRequest>, I>>(base?: I): FromRequest {
    return FromRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<FromRequest>, I>>(object: I): FromRequest {
    const message = createBaseFromRequest();
    message.data = object.data ?? new Uint8Array(0);
    return message;
  },
};

function createBaseFromResponse(): FromResponse {
  return { spec: undefined };
}

export const FromResponse = {
  encode(message: FromResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.spec !== undefined) {
      Spec.encode(message.spec, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FromResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFromResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.spec = Spec.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FromResponse {
    return { spec: isSet(object.spec) ? Spec.fromJSON(object.spec) : undefined };
  },

  toJSON(message: FromResponse): unknown {
    const obj: any = {};
    if (message.spec !== undefined) {
      obj.spec = Spec.toJSON(message.spec);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<FromResponse>, I>>(base?: I): FromResponse {
    return FromResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<FromResponse>, I>>(object: I): FromResponse {
    const message = createBaseFromResponse();
    message.spec = (object.spec !== undefined && object.spec !== null) ? Spec.fromPartial(object.spec) : undefined;
    return message;
  },
};

function createBaseGenRequest(): GenRequest {
  return { spec: undefined };
}

export const GenRequest = {
  encode(message: GenRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.spec !== undefined) {
      Spec.encode(message.spec, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.spec = Spec.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GenRequest {
    return { spec: isSet(object.spec) ? Spec.fromJSON(object.spec) : undefined };
  },

  toJSON(message: GenRequest): unknown {
    const obj: any = {};
    if (message.spec !== undefined) {
      obj.spec = Spec.toJSON(message.spec);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GenRequest>, I>>(base?: I): GenRequest {
    return GenRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GenRequest>, I>>(object: I): GenRequest {
    const message = createBaseGenRequest();
    message.spec = (object.spec !== undefined && object.spec !== null) ? Spec.fromPartial(object.spec) : undefined;
    return message;
  },
};

function createBaseGenResponse(): GenResponse {
  return { data: new Uint8Array(0) };
}

export const GenResponse = {
  encode(message: GenResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.data.length !== 0) {
      writer.uint32(10).bytes(message.data);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.data = reader.bytes();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GenResponse {
    return { data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(0) };
  },

  toJSON(message: GenResponse): unknown {
    const obj: any = {};
    if (message.data.length !== 0) {
      obj.data = base64FromBytes(message.data);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GenResponse>, I>>(base?: I): GenResponse {
    return GenResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GenResponse>, I>>(object: I): GenResponse {
    const message = createBaseGenResponse();
    message.data = object.data ?? new Uint8Array(0);
    return message;
  },
};

function createBaseToRequest(): ToRequest {
  return { spec: undefined };
}

export const ToRequest = {
  encode(message: ToRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.spec !== undefined) {
      Spec.encode(message.spec, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ToRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseToRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.spec = Spec.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ToRequest {
    return { spec: isSet(object.spec) ? Spec.fromJSON(object.spec) : undefined };
  },

  toJSON(message: ToRequest): unknown {
    const obj: any = {};
    if (message.spec !== undefined) {
      obj.spec = Spec.toJSON(message.spec);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ToRequest>, I>>(base?: I): ToRequest {
    return ToRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ToRequest>, I>>(object: I): ToRequest {
    const message = createBaseToRequest();
    message.spec = (object.spec !== undefined && object.spec !== null) ? Spec.fromPartial(object.spec) : undefined;
    return message;
  },
};

function createBaseToResponse(): ToResponse {
  return { data: new Uint8Array(0) };
}

export const ToResponse = {
  encode(message: ToResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.data.length !== 0) {
      writer.uint32(10).bytes(message.data);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ToResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseToResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.data = reader.bytes();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ToResponse {
    return { data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(0) };
  },

  toJSON(message: ToResponse): unknown {
    const obj: any = {};
    if (message.data.length !== 0) {
      obj.data = base64FromBytes(message.data);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ToResponse>, I>>(base?: I): ToResponse {
    return ToResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ToResponse>, I>>(object: I): ToResponse {
    const message = createBaseToResponse();
    message.data = object.data ?? new Uint8Array(0);
    return message;
  },
};

function createBaseSpec(): Spec {
  return {
    name: "",
    source: "",
    version: "",
    displayName: "",
    description: "",
    labels: {},
    types: {},
    functions: {},
    meta: {},
  };
}

export const Spec = {
  encode(message: Spec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.source !== "") {
      writer.uint32(18).string(message.source);
    }
    if (message.version !== "") {
      writer.uint32(26).string(message.version);
    }
    if (message.displayName !== "") {
      writer.uint32(34).string(message.displayName);
    }
    if (message.description !== "") {
      writer.uint32(42).string(message.description);
    }
    Object.entries(message.labels).forEach(([key, value]) => {
      Spec_LabelsEntry.encode({ key: key as any, value }, writer.uint32(50).fork()).ldelim();
    });
    Object.entries(message.types).forEach(([key, value]) => {
      Spec_TypesEntry.encode({ key: key as any, value }, writer.uint32(58).fork()).ldelim();
    });
    Object.entries(message.functions).forEach(([key, value]) => {
      Spec_FunctionsEntry.encode({ key: key as any, value }, writer.uint32(66).fork()).ldelim();
    });
    Object.entries(message.meta).forEach(([key, value]) => {
      Spec_MetaEntry.encode({ key: key as any, value }, writer.uint32(1026).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Spec {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSpec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.source = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.version = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.displayName = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.description = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          const entry6 = Spec_LabelsEntry.decode(reader, reader.uint32());
          if (entry6.value !== undefined) {
            message.labels[entry6.key] = entry6.value;
          }
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          const entry7 = Spec_TypesEntry.decode(reader, reader.uint32());
          if (entry7.value !== undefined) {
            message.types[entry7.key] = entry7.value;
          }
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          const entry8 = Spec_FunctionsEntry.decode(reader, reader.uint32());
          if (entry8.value !== undefined) {
            message.functions[entry8.key] = entry8.value;
          }
          continue;
        case 128:
          if (tag !== 1026) {
            break;
          }

          const entry128 = Spec_MetaEntry.decode(reader, reader.uint32());
          if (entry128.value !== undefined) {
            message.meta[entry128.key] = entry128.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Spec {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      source: isSet(object.source) ? globalThis.String(object.source) : "",
      version: isSet(object.version) ? globalThis.String(object.version) : "",
      displayName: isSet(object.displayName) ? globalThis.String(object.displayName) : "",
      description: isSet(object.description) ? globalThis.String(object.description) : "",
      labels: isObject(object.labels)
        ? Object.entries(object.labels).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
      types: isObject(object.types)
        ? Object.entries(object.types).reduce<{ [key: string]: Type }>((acc, [key, value]) => {
          acc[key] = Type.fromJSON(value);
          return acc;
        }, {})
        : {},
      functions: isObject(object.functions)
        ? Object.entries(object.functions).reduce<{ [key: string]: Function }>((acc, [key, value]) => {
          acc[key] = Function.fromJSON(value);
          return acc;
        }, {})
        : {},
      meta: isObject(object.meta)
        ? Object.entries(object.meta).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
          acc[key] = Any.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Spec): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.source !== "") {
      obj.source = message.source;
    }
    if (message.version !== "") {
      obj.version = message.version;
    }
    if (message.displayName !== "") {
      obj.displayName = message.displayName;
    }
    if (message.description !== "") {
      obj.description = message.description;
    }
    if (message.labels) {
      const entries = Object.entries(message.labels);
      if (entries.length > 0) {
        obj.labels = {};
        entries.forEach(([k, v]) => {
          obj.labels[k] = v;
        });
      }
    }
    if (message.types) {
      const entries = Object.entries(message.types);
      if (entries.length > 0) {
        obj.types = {};
        entries.forEach(([k, v]) => {
          obj.types[k] = Type.toJSON(v);
        });
      }
    }
    if (message.functions) {
      const entries = Object.entries(message.functions);
      if (entries.length > 0) {
        obj.functions = {};
        entries.forEach(([k, v]) => {
          obj.functions[k] = Function.toJSON(v);
        });
      }
    }
    if (message.meta) {
      const entries = Object.entries(message.meta);
      if (entries.length > 0) {
        obj.meta = {};
        entries.forEach(([k, v]) => {
          obj.meta[k] = Any.toJSON(v);
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Spec>, I>>(base?: I): Spec {
    return Spec.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Spec>, I>>(object: I): Spec {
    const message = createBaseSpec();
    message.name = object.name ?? "";
    message.source = object.source ?? "";
    message.version = object.version ?? "";
    message.displayName = object.displayName ?? "";
    message.description = object.description ?? "";
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = globalThis.String(value);
      }
      return acc;
    }, {});
    message.types = Object.entries(object.types ?? {}).reduce<{ [key: string]: Type }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Type.fromPartial(value);
      }
      return acc;
    }, {});
    message.functions = Object.entries(object.functions ?? {}).reduce<{ [key: string]: Function }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = Function.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    message.meta = Object.entries(object.meta ?? {}).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Any.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseSpec_LabelsEntry(): Spec_LabelsEntry {
  return { key: "", value: "" };
}

export const Spec_LabelsEntry = {
  encode(message: Spec_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Spec_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSpec_LabelsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Spec_LabelsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: Spec_LabelsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Spec_LabelsEntry>, I>>(base?: I): Spec_LabelsEntry {
    return Spec_LabelsEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Spec_LabelsEntry>, I>>(object: I): Spec_LabelsEntry {
    const message = createBaseSpec_LabelsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBaseSpec_TypesEntry(): Spec_TypesEntry {
  return { key: "", value: undefined };
}

export const Spec_TypesEntry = {
  encode(message: Spec_TypesEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Type.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Spec_TypesEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSpec_TypesEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Type.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Spec_TypesEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Type.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Spec_TypesEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Type.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Spec_TypesEntry>, I>>(base?: I): Spec_TypesEntry {
    return Spec_TypesEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Spec_TypesEntry>, I>>(object: I): Spec_TypesEntry {
    const message = createBaseSpec_TypesEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Type.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseSpec_FunctionsEntry(): Spec_FunctionsEntry {
  return { key: "", value: undefined };
}

export const Spec_FunctionsEntry = {
  encode(message: Spec_FunctionsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Function.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Spec_FunctionsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSpec_FunctionsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Function.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Spec_FunctionsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Function.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Spec_FunctionsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Function.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Spec_FunctionsEntry>, I>>(base?: I): Spec_FunctionsEntry {
    return Spec_FunctionsEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Spec_FunctionsEntry>, I>>(object: I): Spec_FunctionsEntry {
    const message = createBaseSpec_FunctionsEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null)
      ? Function.fromPartial(object.value)
      : undefined;
    return message;
  },
};

function createBaseSpec_MetaEntry(): Spec_MetaEntry {
  return { key: "", value: undefined };
}

export const Spec_MetaEntry = {
  encode(message: Spec_MetaEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Any.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Spec_MetaEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSpec_MetaEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Any.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Spec_MetaEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Any.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Spec_MetaEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Any.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Spec_MetaEntry>, I>>(base?: I): Spec_MetaEntry {
    return Spec_MetaEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Spec_MetaEntry>, I>>(object: I): Spec_MetaEntry {
    const message = createBaseSpec_MetaEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Any.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseType(): Type {
  return { type: "", fields: {}, methods: {}, genericParameters: {}, constructor: undefined, meta: {} };
}

export const Type = {
  encode(message: Type, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== "") {
      writer.uint32(10).string(message.type);
    }
    Object.entries(message.fields).forEach(([key, value]) => {
      Type_FieldsEntry.encode({ key: key as any, value }, writer.uint32(18).fork()).ldelim();
    });
    Object.entries(message.methods).forEach(([key, value]) => {
      Type_MethodsEntry.encode({ key: key as any, value }, writer.uint32(26).fork()).ldelim();
    });
    Object.entries(message.genericParameters).forEach(([key, value]) => {
      Type_GenericParametersEntry.encode({ key: key as any, value }, writer.uint32(34).fork()).ldelim();
    });
    if (message.constructor !== undefined) {
      Constructor.encode(message.constructor, writer.uint32(42).fork()).ldelim();
    }
    Object.entries(message.meta).forEach(([key, value]) => {
      Type_MetaEntry.encode({ key: key as any, value }, writer.uint32(1026).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Type {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseType();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.type = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          const entry2 = Type_FieldsEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.fields[entry2.key] = entry2.value;
          }
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          const entry3 = Type_MethodsEntry.decode(reader, reader.uint32());
          if (entry3.value !== undefined) {
            message.methods[entry3.key] = entry3.value;
          }
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          const entry4 = Type_GenericParametersEntry.decode(reader, reader.uint32());
          if (entry4.value !== undefined) {
            message.genericParameters[entry4.key] = entry4.value;
          }
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.constructor = Constructor.decode(reader, reader.uint32());
          continue;
        case 128:
          if (tag !== 1026) {
            break;
          }

          const entry128 = Type_MetaEntry.decode(reader, reader.uint32());
          if (entry128.value !== undefined) {
            message.meta[entry128.key] = entry128.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Type {
    return {
      type: isSet(object.type) ? globalThis.String(object.type) : "",
      fields: isObject(object.fields)
        ? Object.entries(object.fields).reduce<{ [key: string]: Field }>((acc, [key, value]) => {
          acc[key] = Field.fromJSON(value);
          return acc;
        }, {})
        : {},
      methods: isObject(object.methods)
        ? Object.entries(object.methods).reduce<{ [key: string]: Function }>((acc, [key, value]) => {
          acc[key] = Function.fromJSON(value);
          return acc;
        }, {})
        : {},
      genericParameters: isObject(object.genericParameters)
        ? Object.entries(object.genericParameters).reduce<{ [key: string]: GenericParameter }>((acc, [key, value]) => {
          acc[key] = GenericParameter.fromJSON(value);
          return acc;
        }, {})
        : {},
      constructor: isSet(object.constructor) ? Constructor.fromJSON(object.constructor) : undefined,
      meta: isObject(object.meta)
        ? Object.entries(object.meta).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
          acc[key] = Any.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Type): unknown {
    const obj: any = {};
    if (message.type !== "") {
      obj.type = message.type;
    }
    if (message.fields) {
      const entries = Object.entries(message.fields);
      if (entries.length > 0) {
        obj.fields = {};
        entries.forEach(([k, v]) => {
          obj.fields[k] = Field.toJSON(v);
        });
      }
    }
    if (message.methods) {
      const entries = Object.entries(message.methods);
      if (entries.length > 0) {
        obj.methods = {};
        entries.forEach(([k, v]) => {
          obj.methods[k] = Function.toJSON(v);
        });
      }
    }
    if (message.genericParameters) {
      const entries = Object.entries(message.genericParameters);
      if (entries.length > 0) {
        obj.genericParameters = {};
        entries.forEach(([k, v]) => {
          obj.genericParameters[k] = GenericParameter.toJSON(v);
        });
      }
    }
    if (message.constructor !== undefined) {
      obj.constructor = Constructor.toJSON(message.constructor);
    }
    if (message.meta) {
      const entries = Object.entries(message.meta);
      if (entries.length > 0) {
        obj.meta = {};
        entries.forEach(([k, v]) => {
          obj.meta[k] = Any.toJSON(v);
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Type>, I>>(base?: I): Type {
    return Type.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Type>, I>>(object: I): Type {
    const message = createBaseType();
    message.type = object.type ?? "";
    message.fields = Object.entries(object.fields ?? {}).reduce<{ [key: string]: Field }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Field.fromPartial(value);
      }
      return acc;
    }, {});
    message.methods = Object.entries(object.methods ?? {}).reduce<{ [key: string]: Function }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Function.fromPartial(value);
      }
      return acc;
    }, {});
    message.genericParameters = Object.entries(object.genericParameters ?? {}).reduce<
      { [key: string]: GenericParameter }
    >((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = GenericParameter.fromPartial(value);
      }
      return acc;
    }, {});
    message.constructor = (object.constructor !== undefined && object.constructor !== null)
      ? Constructor.fromPartial(object.constructor)
      : undefined;
    message.meta = Object.entries(object.meta ?? {}).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Any.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseType_FieldsEntry(): Type_FieldsEntry {
  return { key: "", value: undefined };
}

export const Type_FieldsEntry = {
  encode(message: Type_FieldsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Field.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Type_FieldsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseType_FieldsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Field.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Type_FieldsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Field.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Type_FieldsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Field.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Type_FieldsEntry>, I>>(base?: I): Type_FieldsEntry {
    return Type_FieldsEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Type_FieldsEntry>, I>>(object: I): Type_FieldsEntry {
    const message = createBaseType_FieldsEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Field.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseType_MethodsEntry(): Type_MethodsEntry {
  return { key: "", value: undefined };
}

export const Type_MethodsEntry = {
  encode(message: Type_MethodsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Function.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Type_MethodsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseType_MethodsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Function.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Type_MethodsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Function.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Type_MethodsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Function.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Type_MethodsEntry>, I>>(base?: I): Type_MethodsEntry {
    return Type_MethodsEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Type_MethodsEntry>, I>>(object: I): Type_MethodsEntry {
    const message = createBaseType_MethodsEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null)
      ? Function.fromPartial(object.value)
      : undefined;
    return message;
  },
};

function createBaseType_GenericParametersEntry(): Type_GenericParametersEntry {
  return { key: "", value: undefined };
}

export const Type_GenericParametersEntry = {
  encode(message: Type_GenericParametersEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      GenericParameter.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Type_GenericParametersEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseType_GenericParametersEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = GenericParameter.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Type_GenericParametersEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? GenericParameter.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Type_GenericParametersEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = GenericParameter.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Type_GenericParametersEntry>, I>>(base?: I): Type_GenericParametersEntry {
    return Type_GenericParametersEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Type_GenericParametersEntry>, I>>(object: I): Type_GenericParametersEntry {
    const message = createBaseType_GenericParametersEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null)
      ? GenericParameter.fromPartial(object.value)
      : undefined;
    return message;
  },
};

function createBaseType_MetaEntry(): Type_MetaEntry {
  return { key: "", value: undefined };
}

export const Type_MetaEntry = {
  encode(message: Type_MetaEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Any.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Type_MetaEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseType_MetaEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Any.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Type_MetaEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Any.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Type_MetaEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Any.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Type_MetaEntry>, I>>(base?: I): Type_MetaEntry {
    return Type_MetaEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Type_MetaEntry>, I>>(object: I): Type_MetaEntry {
    const message = createBaseType_MetaEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Any.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseField(): Field {
  return { type: "", readonly: false, meta: {} };
}

export const Field = {
  encode(message: Field, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== "") {
      writer.uint32(10).string(message.type);
    }
    if (message.readonly === true) {
      writer.uint32(16).bool(message.readonly);
    }
    Object.entries(message.meta).forEach(([key, value]) => {
      Field_MetaEntry.encode({ key: key as any, value }, writer.uint32(1026).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Field {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseField();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.type = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.readonly = reader.bool();
          continue;
        case 128:
          if (tag !== 1026) {
            break;
          }

          const entry128 = Field_MetaEntry.decode(reader, reader.uint32());
          if (entry128.value !== undefined) {
            message.meta[entry128.key] = entry128.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Field {
    return {
      type: isSet(object.type) ? globalThis.String(object.type) : "",
      readonly: isSet(object.readonly) ? globalThis.Boolean(object.readonly) : false,
      meta: isObject(object.meta)
        ? Object.entries(object.meta).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
          acc[key] = Any.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Field): unknown {
    const obj: any = {};
    if (message.type !== "") {
      obj.type = message.type;
    }
    if (message.readonly === true) {
      obj.readonly = message.readonly;
    }
    if (message.meta) {
      const entries = Object.entries(message.meta);
      if (entries.length > 0) {
        obj.meta = {};
        entries.forEach(([k, v]) => {
          obj.meta[k] = Any.toJSON(v);
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Field>, I>>(base?: I): Field {
    return Field.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Field>, I>>(object: I): Field {
    const message = createBaseField();
    message.type = object.type ?? "";
    message.readonly = object.readonly ?? false;
    message.meta = Object.entries(object.meta ?? {}).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Any.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseField_MetaEntry(): Field_MetaEntry {
  return { key: "", value: undefined };
}

export const Field_MetaEntry = {
  encode(message: Field_MetaEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Any.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Field_MetaEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseField_MetaEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Any.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Field_MetaEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Any.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Field_MetaEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Any.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Field_MetaEntry>, I>>(base?: I): Field_MetaEntry {
    return Field_MetaEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Field_MetaEntry>, I>>(object: I): Field_MetaEntry {
    const message = createBaseField_MetaEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Any.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseFunction(): Function {
  return { returnType: undefined, parameters: {}, genericParameters: {}, meta: {} };
}

export const Function = {
  encode(message: Function, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.returnType !== undefined) {
      Type.encode(message.returnType, writer.uint32(10).fork()).ldelim();
    }
    Object.entries(message.parameters).forEach(([key, value]) => {
      Function_ParametersEntry.encode({ key: key as any, value }, writer.uint32(18).fork()).ldelim();
    });
    Object.entries(message.genericParameters).forEach(([key, value]) => {
      Function_GenericParametersEntry.encode({ key: key as any, value }, writer.uint32(26).fork()).ldelim();
    });
    Object.entries(message.meta).forEach(([key, value]) => {
      Function_MetaEntry.encode({ key: key as any, value }, writer.uint32(1026).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Function {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFunction();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.returnType = Type.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          const entry2 = Function_ParametersEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.parameters[entry2.key] = entry2.value;
          }
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          const entry3 = Function_GenericParametersEntry.decode(reader, reader.uint32());
          if (entry3.value !== undefined) {
            message.genericParameters[entry3.key] = entry3.value;
          }
          continue;
        case 128:
          if (tag !== 1026) {
            break;
          }

          const entry128 = Function_MetaEntry.decode(reader, reader.uint32());
          if (entry128.value !== undefined) {
            message.meta[entry128.key] = entry128.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Function {
    return {
      returnType: isSet(object.returnType) ? Type.fromJSON(object.returnType) : undefined,
      parameters: isObject(object.parameters)
        ? Object.entries(object.parameters).reduce<{ [key: string]: Type }>((acc, [key, value]) => {
          acc[key] = Type.fromJSON(value);
          return acc;
        }, {})
        : {},
      genericParameters: isObject(object.genericParameters)
        ? Object.entries(object.genericParameters).reduce<{ [key: string]: GenericParameter }>((acc, [key, value]) => {
          acc[key] = GenericParameter.fromJSON(value);
          return acc;
        }, {})
        : {},
      meta: isObject(object.meta)
        ? Object.entries(object.meta).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
          acc[key] = Any.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Function): unknown {
    const obj: any = {};
    if (message.returnType !== undefined) {
      obj.returnType = Type.toJSON(message.returnType);
    }
    if (message.parameters) {
      const entries = Object.entries(message.parameters);
      if (entries.length > 0) {
        obj.parameters = {};
        entries.forEach(([k, v]) => {
          obj.parameters[k] = Type.toJSON(v);
        });
      }
    }
    if (message.genericParameters) {
      const entries = Object.entries(message.genericParameters);
      if (entries.length > 0) {
        obj.genericParameters = {};
        entries.forEach(([k, v]) => {
          obj.genericParameters[k] = GenericParameter.toJSON(v);
        });
      }
    }
    if (message.meta) {
      const entries = Object.entries(message.meta);
      if (entries.length > 0) {
        obj.meta = {};
        entries.forEach(([k, v]) => {
          obj.meta[k] = Any.toJSON(v);
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Function>, I>>(base?: I): Function {
    return Function.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Function>, I>>(object: I): Function {
    const message = createBaseFunction();
    message.returnType = (object.returnType !== undefined && object.returnType !== null)
      ? Type.fromPartial(object.returnType)
      : undefined;
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{ [key: string]: Type }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = Type.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    message.genericParameters = Object.entries(object.genericParameters ?? {}).reduce<
      { [key: string]: GenericParameter }
    >((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = GenericParameter.fromPartial(value);
      }
      return acc;
    }, {});
    message.meta = Object.entries(object.meta ?? {}).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Any.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseFunction_ParametersEntry(): Function_ParametersEntry {
  return { key: "", value: undefined };
}

export const Function_ParametersEntry = {
  encode(message: Function_ParametersEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Type.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Function_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFunction_ParametersEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Type.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Function_ParametersEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Type.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Function_ParametersEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Type.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Function_ParametersEntry>, I>>(base?: I): Function_ParametersEntry {
    return Function_ParametersEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Function_ParametersEntry>, I>>(object: I): Function_ParametersEntry {
    const message = createBaseFunction_ParametersEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Type.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseFunction_GenericParametersEntry(): Function_GenericParametersEntry {
  return { key: "", value: undefined };
}

export const Function_GenericParametersEntry = {
  encode(message: Function_GenericParametersEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      GenericParameter.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Function_GenericParametersEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFunction_GenericParametersEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = GenericParameter.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Function_GenericParametersEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? GenericParameter.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Function_GenericParametersEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = GenericParameter.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Function_GenericParametersEntry>, I>>(base?: I): Function_GenericParametersEntry {
    return Function_GenericParametersEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Function_GenericParametersEntry>, I>>(
    object: I,
  ): Function_GenericParametersEntry {
    const message = createBaseFunction_GenericParametersEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null)
      ? GenericParameter.fromPartial(object.value)
      : undefined;
    return message;
  },
};

function createBaseFunction_MetaEntry(): Function_MetaEntry {
  return { key: "", value: undefined };
}

export const Function_MetaEntry = {
  encode(message: Function_MetaEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Any.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Function_MetaEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFunction_MetaEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Any.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Function_MetaEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Any.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Function_MetaEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Any.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Function_MetaEntry>, I>>(base?: I): Function_MetaEntry {
    return Function_MetaEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Function_MetaEntry>, I>>(object: I): Function_MetaEntry {
    const message = createBaseFunction_MetaEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Any.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseGenericParameter(): GenericParameter {
  return { modifiers: [], meta: {} };
}

export const GenericParameter = {
  encode(message: GenericParameter, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.modifiers) {
      Modifier.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    Object.entries(message.meta).forEach(([key, value]) => {
      GenericParameter_MetaEntry.encode({ key: key as any, value }, writer.uint32(1026).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenericParameter {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenericParameter();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.modifiers.push(Modifier.decode(reader, reader.uint32()));
          continue;
        case 128:
          if (tag !== 1026) {
            break;
          }

          const entry128 = GenericParameter_MetaEntry.decode(reader, reader.uint32());
          if (entry128.value !== undefined) {
            message.meta[entry128.key] = entry128.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GenericParameter {
    return {
      modifiers: globalThis.Array.isArray(object?.modifiers)
        ? object.modifiers.map((e: any) => Modifier.fromJSON(e))
        : [],
      meta: isObject(object.meta)
        ? Object.entries(object.meta).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
          acc[key] = Any.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: GenericParameter): unknown {
    const obj: any = {};
    if (message.modifiers?.length) {
      obj.modifiers = message.modifiers.map((e) => Modifier.toJSON(e));
    }
    if (message.meta) {
      const entries = Object.entries(message.meta);
      if (entries.length > 0) {
        obj.meta = {};
        entries.forEach(([k, v]) => {
          obj.meta[k] = Any.toJSON(v);
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GenericParameter>, I>>(base?: I): GenericParameter {
    return GenericParameter.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GenericParameter>, I>>(object: I): GenericParameter {
    const message = createBaseGenericParameter();
    message.modifiers = object.modifiers?.map((e) => Modifier.fromPartial(e)) || [];
    message.meta = Object.entries(object.meta ?? {}).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Any.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseGenericParameter_MetaEntry(): GenericParameter_MetaEntry {
  return { key: "", value: undefined };
}

export const GenericParameter_MetaEntry = {
  encode(message: GenericParameter_MetaEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Any.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenericParameter_MetaEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenericParameter_MetaEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Any.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GenericParameter_MetaEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Any.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: GenericParameter_MetaEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Any.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GenericParameter_MetaEntry>, I>>(base?: I): GenericParameter_MetaEntry {
    return GenericParameter_MetaEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GenericParameter_MetaEntry>, I>>(object: I): GenericParameter_MetaEntry {
    const message = createBaseGenericParameter_MetaEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Any.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseModifier(): Modifier {
  return { meta: {} };
}

export const Modifier = {
  encode(message: Modifier, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.meta).forEach(([key, value]) => {
      Modifier_MetaEntry.encode({ key: key as any, value }, writer.uint32(1026).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Modifier {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseModifier();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 128:
          if (tag !== 1026) {
            break;
          }

          const entry128 = Modifier_MetaEntry.decode(reader, reader.uint32());
          if (entry128.value !== undefined) {
            message.meta[entry128.key] = entry128.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Modifier {
    return {
      meta: isObject(object.meta)
        ? Object.entries(object.meta).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
          acc[key] = Any.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Modifier): unknown {
    const obj: any = {};
    if (message.meta) {
      const entries = Object.entries(message.meta);
      if (entries.length > 0) {
        obj.meta = {};
        entries.forEach(([k, v]) => {
          obj.meta[k] = Any.toJSON(v);
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Modifier>, I>>(base?: I): Modifier {
    return Modifier.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Modifier>, I>>(object: I): Modifier {
    const message = createBaseModifier();
    message.meta = Object.entries(object.meta ?? {}).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Any.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseModifier_MetaEntry(): Modifier_MetaEntry {
  return { key: "", value: undefined };
}

export const Modifier_MetaEntry = {
  encode(message: Modifier_MetaEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Any.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Modifier_MetaEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseModifier_MetaEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Any.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Modifier_MetaEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Any.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Modifier_MetaEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Any.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Modifier_MetaEntry>, I>>(base?: I): Modifier_MetaEntry {
    return Modifier_MetaEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Modifier_MetaEntry>, I>>(object: I): Modifier_MetaEntry {
    const message = createBaseModifier_MetaEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Any.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseConstructor(): Constructor {
  return { parameters: {}, meta: {} };
}

export const Constructor = {
  encode(message: Constructor, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.parameters).forEach(([key, value]) => {
      Constructor_ParametersEntry.encode({ key: key as any, value }, writer.uint32(10).fork()).ldelim();
    });
    Object.entries(message.meta).forEach(([key, value]) => {
      Constructor_MetaEntry.encode({ key: key as any, value }, writer.uint32(1026).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Constructor {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConstructor();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          const entry1 = Constructor_ParametersEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.parameters[entry1.key] = entry1.value;
          }
          continue;
        case 128:
          if (tag !== 1026) {
            break;
          }

          const entry128 = Constructor_MetaEntry.decode(reader, reader.uint32());
          if (entry128.value !== undefined) {
            message.meta[entry128.key] = entry128.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Constructor {
    return {
      parameters: isObject(object.parameters)
        ? Object.entries(object.parameters).reduce<{ [key: string]: Type }>((acc, [key, value]) => {
          acc[key] = Type.fromJSON(value);
          return acc;
        }, {})
        : {},
      meta: isObject(object.meta)
        ? Object.entries(object.meta).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
          acc[key] = Any.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Constructor): unknown {
    const obj: any = {};
    if (message.parameters) {
      const entries = Object.entries(message.parameters);
      if (entries.length > 0) {
        obj.parameters = {};
        entries.forEach(([k, v]) => {
          obj.parameters[k] = Type.toJSON(v);
        });
      }
    }
    if (message.meta) {
      const entries = Object.entries(message.meta);
      if (entries.length > 0) {
        obj.meta = {};
        entries.forEach(([k, v]) => {
          obj.meta[k] = Any.toJSON(v);
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Constructor>, I>>(base?: I): Constructor {
    return Constructor.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Constructor>, I>>(object: I): Constructor {
    const message = createBaseConstructor();
    message.parameters = Object.entries(object.parameters ?? {}).reduce<{ [key: string]: Type }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = Type.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    message.meta = Object.entries(object.meta ?? {}).reduce<{ [key: string]: Any }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Any.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseConstructor_ParametersEntry(): Constructor_ParametersEntry {
  return { key: "", value: undefined };
}

export const Constructor_ParametersEntry = {
  encode(message: Constructor_ParametersEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Type.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Constructor_ParametersEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConstructor_ParametersEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Type.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Constructor_ParametersEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Type.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Constructor_ParametersEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Type.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Constructor_ParametersEntry>, I>>(base?: I): Constructor_ParametersEntry {
    return Constructor_ParametersEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Constructor_ParametersEntry>, I>>(object: I): Constructor_ParametersEntry {
    const message = createBaseConstructor_ParametersEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Type.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseConstructor_MetaEntry(): Constructor_MetaEntry {
  return { key: "", value: undefined };
}

export const Constructor_MetaEntry = {
  encode(message: Constructor_MetaEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Any.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Constructor_MetaEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConstructor_MetaEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = Any.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Constructor_MetaEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Any.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Constructor_MetaEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Any.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Constructor_MetaEntry>, I>>(base?: I): Constructor_MetaEntry {
    return Constructor_MetaEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Constructor_MetaEntry>, I>>(object: I): Constructor_MetaEntry {
    const message = createBaseConstructor_MetaEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Any.fromPartial(object.value) : undefined;
    return message;
  },
};

export interface UmlService {
  From(request: Observable<FromRequest>): Promise<FromResponse>;
  Gen(request: GenRequest): Observable<GenResponse>;
  To(request: ToRequest): Observable<ToResponse>;
}

export const UmlServiceServiceName = "unmango.dev.tdl.v1alpha1.UmlService";
export class UmlServiceClientImpl implements UmlService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || UmlServiceServiceName;
    this.rpc = rpc;
    this.From = this.From.bind(this);
    this.Gen = this.Gen.bind(this);
    this.To = this.To.bind(this);
  }
  From(request: Observable<FromRequest>): Promise<FromResponse> {
    const data = request.pipe(map((request) => FromRequest.encode(request).finish()));
    const promise = this.rpc.clientStreamingRequest(this.service, "From", data);
    return promise.then((data) => FromResponse.decode(_m0.Reader.create(data)));
  }

  Gen(request: GenRequest): Observable<GenResponse> {
    const data = GenRequest.encode(request).finish();
    const result = this.rpc.serverStreamingRequest(this.service, "Gen", data);
    return result.pipe(map((data) => GenResponse.decode(_m0.Reader.create(data))));
  }

  To(request: ToRequest): Observable<ToResponse> {
    const data = ToRequest.encode(request).finish();
    const result = this.rpc.serverStreamingRequest(this.service, "To", data);
    return result.pipe(map((data) => ToResponse.decode(_m0.Reader.create(data))));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
  clientStreamingRequest(service: string, method: string, data: Observable<Uint8Array>): Promise<Uint8Array>;
  serverStreamingRequest(service: string, method: string, data: Uint8Array): Observable<Uint8Array>;
  bidirectionalStreamingRequest(service: string, method: string, data: Observable<Uint8Array>): Observable<Uint8Array>;
}

function bytesFromBase64(b64: string): Uint8Array {
  if ((globalThis as any).Buffer) {
    return Uint8Array.from(globalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = globalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if ((globalThis as any).Buffer) {
    return globalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(globalThis.String.fromCharCode(byte));
    });
    return globalThis.btoa(bin.join(""));
  }
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
