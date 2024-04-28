/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Observable } from "rxjs";
import { map } from "rxjs/operators";

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
  repository: string;
  version: string;
  displayName: string;
  description: string;
  tags: string[];
  types: { [key: string]: Type };
}

export interface Spec_TypesEntry {
  key: string;
  value: Type | undefined;
}

export interface Type {
  name: string;
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
  return { name: "", repository: "", version: "", displayName: "", description: "", tags: [], types: {} };
}

export const Spec = {
  encode(message: Spec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.repository !== "") {
      writer.uint32(18).string(message.repository);
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
    for (const v of message.tags) {
      writer.uint32(50).string(v!);
    }
    Object.entries(message.types).forEach(([key, value]) => {
      Spec_TypesEntry.encode({ key: key as any, value }, writer.uint32(58).fork()).ldelim();
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

          message.repository = reader.string();
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

          message.tags.push(reader.string());
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
      repository: isSet(object.repository) ? globalThis.String(object.repository) : "",
      version: isSet(object.version) ? globalThis.String(object.version) : "",
      displayName: isSet(object.displayName) ? globalThis.String(object.displayName) : "",
      description: isSet(object.description) ? globalThis.String(object.description) : "",
      tags: globalThis.Array.isArray(object?.tags) ? object.tags.map((e: any) => globalThis.String(e)) : [],
      types: isObject(object.types)
        ? Object.entries(object.types).reduce<{ [key: string]: Type }>((acc, [key, value]) => {
          acc[key] = Type.fromJSON(value);
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
    if (message.repository !== "") {
      obj.repository = message.repository;
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
    if (message.tags?.length) {
      obj.tags = message.tags;
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
    return obj;
  },

  create<I extends Exact<DeepPartial<Spec>, I>>(base?: I): Spec {
    return Spec.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Spec>, I>>(object: I): Spec {
    const message = createBaseSpec();
    message.name = object.name ?? "";
    message.repository = object.repository ?? "";
    message.version = object.version ?? "";
    message.displayName = object.displayName ?? "";
    message.description = object.description ?? "";
    message.tags = object.tags?.map((e) => e) || [];
    message.types = Object.entries(object.types ?? {}).reduce<{ [key: string]: Type }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Type.fromPartial(value);
      }
      return acc;
    }, {});
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

function createBaseType(): Type {
  return { name: "" };
}

export const Type = {
  encode(message: Type, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
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

          message.name = reader.string();
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
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: Type): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Type>, I>>(base?: I): Type {
    return Type.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Type>, I>>(object: I): Type {
    const message = createBaseType();
    message.name = object.name ?? "";
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
