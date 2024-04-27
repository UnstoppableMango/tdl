/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "unmango.dev.tdl.v1alpha1";

export interface PullRequest {
  actor: string;
}

export interface PullResponse {
  uml: string;
}

function createBasePullRequest(): PullRequest {
  return { actor: "" };
}

export const PullRequest = {
  encode(message: PullRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.actor !== "") {
      writer.uint32(10).string(message.actor);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PullRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePullRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.actor = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PullRequest {
    return { actor: isSet(object.actor) ? globalThis.String(object.actor) : "" };
  },

  toJSON(message: PullRequest): unknown {
    const obj: any = {};
    if (message.actor !== "") {
      obj.actor = message.actor;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<PullRequest>, I>>(base?: I): PullRequest {
    return PullRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<PullRequest>, I>>(object: I): PullRequest {
    const message = createBasePullRequest();
    message.actor = object.actor ?? "";
    return message;
  },
};

function createBasePullResponse(): PullResponse {
  return { uml: "" };
}

export const PullResponse = {
  encode(message: PullResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.uml !== "") {
      writer.uint32(10).string(message.uml);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PullResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePullResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.uml = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PullResponse {
    return { uml: isSet(object.uml) ? globalThis.String(object.uml) : "" };
  },

  toJSON(message: PullResponse): unknown {
    const obj: any = {};
    if (message.uml !== "") {
      obj.uml = message.uml;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<PullResponse>, I>>(base?: I): PullResponse {
    return PullResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<PullResponse>, I>>(object: I): PullResponse {
    const message = createBasePullResponse();
    message.uml = object.uml ?? "";
    return message;
  },
};

export interface UmlService {
  Pull(request: PullRequest): Promise<PullResponse>;
}

export const UmlServiceServiceName = "unmango.dev.tdl.v1alpha1.UmlService";
export class UmlServiceClientImpl implements UmlService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || UmlServiceServiceName;
    this.rpc = rpc;
    this.Pull = this.Pull.bind(this);
  }
  Pull(request: PullRequest): Promise<PullResponse> {
    const data = PullRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Pull", data);
    return promise.then((data) => PullResponse.decode(_m0.Reader.create(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
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

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
