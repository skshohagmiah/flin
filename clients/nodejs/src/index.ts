import * as net from 'net';
import { EventEmitter } from 'events';
import * as http from 'http';

// Protocol constants
const OpCode = {
  SET: 0x01,
  GET: 0x02,
  DEL: 0x03,
  EXISTS: 0x04,
  INCR: 0x05,
  DECR: 0x06,
  MSET: 0x10,
  MGET: 0x11,
  MDEL: 0x12,
} as const;

const Status = {
  OK: 0x00,
  ERROR: 0x01,
  NOT_FOUND: 0x02,
  MULTI_VALUE: 0x03,
} as const;

export interface FlinOptions {
  host?: string;
  port?: number;
  minConnections?: number;
  maxConnections?: number;
  connectionTimeout?: number;
  requestTimeout?: number;
}

export interface ClusterOptions {
  httpAddresses: string[]; // e.g., ['localhost:7080', 'localhost:7081']
  minConnectionsPerNode?: number;
  maxConnectionsPerNode?: number;
  connectionTimeout?: number;
  requestTimeout?: number;
  refreshInterval?: number;
  partitionCount?: number;
}

interface Node {
  id: string;
  address: string;
}

interface Partition {
  id: number;
  primaryNode: string;
  replicaNodes: string[];
}

interface ClusterTopology {
  nodes: Node[];
  partitionMap: Map<number, Partition>;
  lastUpdate: Date;
}

// Simple client for single-node connections
export class FlinClient extends EventEmitter {
  private pool: ConnectionPool;
  private options: Required<FlinOptions>;

  constructor(options: FlinOptions = {}) {
    super();
    
    this.options = {
      host: options.host || 'localhost',
      port: options.port || 7380,
      minConnections: options.minConnections || 5,
      maxConnections: options.maxConnections || 50,
      connectionTimeout: options.connectionTimeout || 5000,
      requestTimeout: options.requestTimeout || 10000,
    };

    this.pool = new ConnectionPool(this.options);
  }

  async set(key: string, value: Buffer | string): Promise<void> {
    const conn = await this.pool.acquire();
    try {
      const valueBuffer = typeof value === 'string' ? Buffer.from(value) : value;
      await conn.set(key, valueBuffer);
    } finally {
      this.pool.release(conn);
    }
  }

  async get(key: string): Promise<Buffer | null> {
    const conn = await this.pool.acquire();
    try {
      return await conn.get(key);
    } finally {
      this.pool.release(conn);
    }
  }

  async getString(key: string): Promise<string | null> {
    const value = await this.get(key);
    return value ? value.toString('utf8') : null;
  }

  async delete(key: string): Promise<void> {
    const conn = await this.pool.acquire();
    try {
      await conn.delete(key);
    } finally {
      this.pool.release(conn);
    }
  }

  async exists(key: string): Promise<boolean> {
    const conn = await this.pool.acquire();
    try {
      return await conn.exists(key);
    } finally {
      this.pool.release(conn);
    }
  }

  async incr(key: string): Promise<bigint> {
    const conn = await this.pool.acquire();
    try {
      return await conn.incr(key);
    } finally {
      this.pool.release(conn);
    }
  }

  async decr(key: string): Promise<bigint> {
    const conn = await this.pool.acquire();
    try {
      return await conn.decr(key);
    } finally {
      this.pool.release(conn);
    }
  }

  async mset(entries: Array<[string, Buffer | string]>): Promise<void> {
    const conn = await this.pool.acquire();
    try {
      const keys = entries.map(([k]) => k);
      const values = entries.map(([, v]) => 
        typeof v === 'string' ? Buffer.from(v) : v
      );
      await conn.mset(keys, values);
    } finally {
      this.pool.release(conn);
    }
  }

  async mget(keys: string[]): Promise<Array<Buffer | null>> {
    const conn = await this.pool.acquire();
    try {
      return await conn.mget(keys);
    } finally {
      this.pool.release(conn);
    }
  }

  async mdelete(keys: string[]): Promise<void> {
    const conn = await this.pool.acquire();
    try {
      await conn.mdelete(keys);
    } finally {
      this.pool.release(conn);
    }
  }

  async close(): Promise<void> {
    await this.pool.close();
  }
}

// Cluster-aware client with client-side partitioning
export class ClusterClient extends EventEmitter {
  private topology: ClusterTopology;
  private pools: Map<string, ConnectionPool>;
  private httpAddresses: string[];
  private options: Required<ClusterOptions>;
  private refreshInterval: NodeJS.Timeout | null = null;

  constructor(options: ClusterOptions) {
    super();

    if (!options.httpAddresses || options.httpAddresses.length === 0) {
      throw new Error('At least one HTTP address is required');
    }

    this.options = {
      httpAddresses: options.httpAddresses,
      minConnectionsPerNode: options.minConnectionsPerNode || 2,
      maxConnectionsPerNode: options.maxConnectionsPerNode || 10,
      connectionTimeout: options.connectionTimeout || 5000,
      requestTimeout: options.requestTimeout || 10000,
      refreshInterval: options.refreshInterval || 30000,
      partitionCount: options.partitionCount || 64,
    };

    this.httpAddresses = this.options.httpAddresses;
    this.pools = new Map();
    this.topology = {
      nodes: [],
      partitionMap: new Map(),
      lastUpdate: new Date(),
    };
  }

  async initialize(): Promise<void> {
    await this.refreshTopology();
    
    // Start background topology refresh
    this.refreshInterval = setInterval(() => {
      this.refreshTopology().catch(err => {
        this.emit('error', new Error(`Topology refresh failed: ${err.message}`));
      });
    }, this.options.refreshInterval);
  }

  private async refreshTopology(): Promise<void> {
    let lastError: Error | null = null;

    for (const httpAddr of this.httpAddresses) {
      try {
        const url = `http://${httpAddr}/cluster`;
        const data = await this.httpGet(url);
        const response = JSON.parse(data);

        // Update nodes
        this.topology.nodes = response.cluster.nodes.map((n: any) => ({
          id: n.id,
          address: n.ip,
        }));

        // Update partition map
        this.topology.partitionMap.clear();
        for (const [partId, part] of Object.entries(response.cluster.partition_map.partitions as any)) {
          const id = parseInt(partId);
          this.topology.partitionMap.set(id, {
            id,
            primaryNode: part.primary_node,
            replicaNodes: part.replica_nodes || [],
          });
        }

        this.topology.lastUpdate = new Date();

        // Ensure connection pools for all nodes
        await this.ensureConnectionPools();

        return;
      } catch (err) {
        lastError = err as Error;
      }
    }

    throw new Error(`Failed to refresh topology: ${lastError?.message}`);
  }

  private httpGet(url: string): Promise<string> {
    return new Promise((resolve, reject) => {
      http.get(url, (res) => {
        let data = '';
        res.on('data', (chunk) => data += chunk);
        res.on('end', () => {
          if (res.statusCode === 200) {
            resolve(data);
          } else {
            reject(new Error(`HTTP ${res.statusCode}`));
          }
        });
      }).on('error', reject);
    });
  }

  private async ensureConnectionPools(): Promise<void> {
    for (const node of this.topology.nodes) {
      if (!this.pools.has(node.id)) {
        const [host, port] = node.address.split(':');
        const pool = new ConnectionPool({
          host,
          port: parseInt(port),
          minConnections: this.options.minConnectionsPerNode,
          maxConnections: this.options.maxConnectionsPerNode,
          connectionTimeout: this.options.connectionTimeout,
          requestTimeout: this.options.requestTimeout,
        });
        this.pools.set(node.id, pool);
      }
    }
  }

  private getPartitionForKey(key: string): number {
    // FNV-1a hash
    let hash = 2166136261;
    for (let i = 0; i < key.length; i++) {
      hash ^= key.charCodeAt(i);
      hash = Math.imul(hash, 16777619);
    }
    return (hash >>> 0) % this.options.partitionCount;
  }

  private async getConnectionForKey(key: string): Promise<{ conn: Connection; nodeId: string }> {
    const partitionId = this.getPartitionForKey(key);
    const partition = this.topology.partitionMap.get(partitionId);

    if (!partition) {
      throw new Error(`Partition ${partitionId} not found in topology`);
    }

    // Try primary node first
    let pool = this.pools.get(partition.primaryNode);
    if (pool) {
      try {
        const conn = await pool.acquire();
        return { conn, nodeId: partition.primaryNode };
      } catch (err) {
        // Primary failed, try replicas
      }
    }

    // Try replicas
    for (const replicaNode of partition.replicaNodes) {
      pool = this.pools.get(replicaNode);
      if (pool) {
        try {
          const conn = await pool.acquire();
          return { conn, nodeId: replicaNode };
        } catch (err) {
          continue;
        }
      }
    }

    throw new Error(`No available node for partition ${partitionId}`);
  }

  private releaseConnection(nodeId: string, conn: Connection): void {
    const pool = this.pools.get(nodeId);
    if (pool) {
      pool.release(conn);
    }
  }

  async set(key: string, value: Buffer | string): Promise<void> {
    const { conn, nodeId } = await this.getConnectionForKey(key);
    try {
      const valueBuffer = typeof value === 'string' ? Buffer.from(value) : value;
      await conn.set(key, valueBuffer);
    } finally {
      this.releaseConnection(nodeId, conn);
    }
  }

  async get(key: string): Promise<Buffer | null> {
    const { conn, nodeId } = await this.getConnectionForKey(key);
    try {
      return await conn.get(key);
    } finally {
      this.releaseConnection(nodeId, conn);
    }
  }

  async getString(key: string): Promise<string | null> {
    const value = await this.get(key);
    return value ? value.toString('utf8') : null;
  }

  async delete(key: string): Promise<void> {
    const { conn, nodeId } = await this.getConnectionForKey(key);
    try {
      await conn.delete(key);
    } finally {
      this.releaseConnection(nodeId, conn);
    }
  }

  async exists(key: string): Promise<boolean> {
    const { conn, nodeId } = await this.getConnectionForKey(key);
    try {
      return await conn.exists(key);
    } finally {
      this.releaseConnection(nodeId, conn);
    }
  }

  async incr(key: string): Promise<bigint> {
    const { conn, nodeId } = await this.getConnectionForKey(key);
    try {
      return await conn.incr(key);
    } finally {
      this.releaseConnection(nodeId, conn);
    }
  }

  async decr(key: string): Promise<bigint> {
    const { conn, nodeId } = await this.getConnectionForKey(key);
    try {
      return await conn.decr(key);
    } finally {
      this.releaseConnection(nodeId, conn);
    }
  }

  async mset(entries: Array<[string, Buffer | string]>): Promise<void> {
    // Group by node
    const nodeGroups = new Map<string, Array<[string, Buffer]>>();
    
    for (const [key, value] of entries) {
      const partitionId = this.getPartitionForKey(key);
      const partition = this.topology.partitionMap.get(partitionId);
      if (!partition) continue;

      const valueBuffer = typeof value === 'string' ? Buffer.from(value) : value;
      const group = nodeGroups.get(partition.primaryNode) || [];
      group.push([key, valueBuffer]);
      nodeGroups.set(partition.primaryNode, group);
    }

    // Send to each node in parallel
    await Promise.all(
      Array.from(nodeGroups.entries()).map(async ([nodeId, group]) => {
        const pool = this.pools.get(nodeId);
        if (!pool) return;

        const conn = await pool.acquire();
        try {
          const keys = group.map(([k]) => k);
          const values = group.map(([, v]) => v);
          await conn.mset(keys, values);
        } finally {
          pool.release(conn);
        }
      })
    );
  }

  async mget(keys: string[]): Promise<Array<Buffer | null>> {
    // Group by node
    const nodeGroups = new Map<string, number[]>();
    
    for (let i = 0; i < keys.length; i++) {
      const partitionId = this.getPartitionForKey(keys[i]);
      const partition = this.topology.partitionMap.get(partitionId);
      if (!partition) continue;

      const group = nodeGroups.get(partition.primaryNode) || [];
      group.push(i);
      nodeGroups.set(partition.primaryNode, group);
    }

    const results: Array<Buffer | null> = new Array(keys.length).fill(null);

    // Fetch from each node in parallel
    await Promise.all(
      Array.from(nodeGroups.entries()).map(async ([nodeId, indices]) => {
        const pool = this.pools.get(nodeId);
        if (!pool) return;

        const conn = await pool.acquire();
        try {
          const batchKeys = indices.map(i => keys[i]);
          const values = await conn.mget(batchKeys);
          
          for (let i = 0; i < indices.length; i++) {
            results[indices[i]] = values[i];
          }
        } finally {
          pool.release(conn);
        }
      })
    );

    return results;
  }

  async mdelete(keys: string[]): Promise<void> {
    // Group by node
    const nodeGroups = new Map<string, string[]>();
    
    for (const key of keys) {
      const partitionId = this.getPartitionForKey(key);
      const partition = this.topology.partitionMap.get(partitionId);
      if (!partition) continue;

      const group = nodeGroups.get(partition.primaryNode) || [];
      group.push(key);
      nodeGroups.set(partition.primaryNode, group);
    }

    // Send to each node in parallel
    await Promise.all(
      Array.from(nodeGroups.entries()).map(async ([nodeId, batchKeys]) => {
        const pool = this.pools.get(nodeId);
        if (!pool) return;

        const conn = await pool.acquire();
        try {
          await conn.mdelete(batchKeys);
        } finally {
          pool.release(conn);
        }
      })
    );
  }

  getTopology(): ClusterTopology {
    return {
      nodes: [...this.topology.nodes],
      partitionMap: new Map(this.topology.partitionMap),
      lastUpdate: this.topology.lastUpdate,
    };
  }

  async close(): Promise<void> {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
    }

    await Promise.all(
      Array.from(this.pools.values()).map(pool => pool.close())
    );
  }
}

class Connection {
  private socket: net.Socket;
  private options: Required<FlinOptions>;
  private responseQueue: Array<{
    resolve: (data: any) => void;
    reject: (err: Error) => void;
  }> = [];
  private buffer: Buffer = Buffer.alloc(0);

  constructor(options: Required<FlinOptions>) {
    this.options = options;
    this.socket = new net.Socket();
    this.socket.setNoDelay(true);
    this.socket.setKeepAlive(true, 30000);
    
    this.socket.on('data', (data) => this.handleData(data));
    this.socket.on('error', (err) => this.handleError(err));
  }

  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.socket.destroy();
        reject(new Error('Connection timeout'));
      }, this.options.connectionTimeout);

      this.socket.connect(this.options.port, this.options.host, () => {
        clearTimeout(timeout);
        resolve();
      });

      this.socket.once('error', (err) => {
        clearTimeout(timeout);
        reject(err);
      });
    });
  }

  private handleData(data: Buffer): void {
    this.buffer = Buffer.concat([this.buffer, data]);
    this.processResponses();
  }

  private processResponses(): void {
    while (this.buffer.length >= 5) {
      const status = this.buffer[0];
      const payloadLen = this.buffer.readUInt32BE(1);
      const totalLen = 5 + payloadLen;

      if (this.buffer.length < totalLen) {
        return;
      }

      const payload = this.buffer.slice(5, totalLen);
      this.buffer = this.buffer.slice(totalLen);

      const handler = this.responseQueue.shift();
      if (!handler) continue;

      try {
        if (status === Status.OK) {
          handler.resolve(payload.length > 0 ? payload : null);
        } else if (status === Status.ERROR) {
          handler.reject(new Error(payload.toString('utf8')));
        } else if (status === Status.NOT_FOUND) {
          handler.resolve(null);
        } else if (status === Status.MULTI_VALUE) {
          handler.resolve(this.parseMultiValue(payload));
        } else {
          handler.reject(new Error(`Unknown status: ${status}`));
        }
      } catch (err) {
        handler.reject(err as Error);
      }
    }
  }

  private parseMultiValue(payload: Buffer): Array<Buffer | null> {
    const count = payload.readUInt16BE(0);
    const values: Array<Buffer | null> = [];
    let pos = 2;

    for (let i = 0; i < count; i++) {
      const valueLen = payload.readUInt32BE(pos);
      pos += 4;
      
      if (valueLen === 0) {
        values.push(null);
      } else {
        values.push(payload.slice(pos, pos + valueLen));
        pos += valueLen;
      }
    }

    return values;
  }

  private handleError(err: Error): void {
    while (this.responseQueue.length > 0) {
      const handler = this.responseQueue.shift();
      handler?.reject(err);
    }
  }

  private async sendRequest(request: Buffer): Promise<any> {
    return new Promise((resolve, reject) => {
      this.responseQueue.push({ resolve, reject });
      
      const timeout = setTimeout(() => {
        const idx = this.responseQueue.findIndex(h => h.resolve === resolve);
        if (idx >= 0) {
          this.responseQueue.splice(idx, 1);
          reject(new Error('Request timeout'));
        }
      }, this.options.requestTimeout);

      this.socket.write(request, (err) => {
        if (err) {
          clearTimeout(timeout);
          const idx = this.responseQueue.findIndex(h => h.resolve === resolve);
          if (idx >= 0) {
            this.responseQueue.splice(idx, 1);
          }
          reject(err);
        }
      });
    });
  }

  async set(key: string, value: Buffer): Promise<void> {
    const request = encodeSetRequest(key, value);
    await this.sendRequest(request);
  }

  async get(key: string): Promise<Buffer | null> {
    const request = encodeGetRequest(key);
    return await this.sendRequest(request);
  }

  async delete(key: string): Promise<void> {
    const request = encodeDeleteRequest(key);
    await this.sendRequest(request);
  }

  async exists(key: string): Promise<boolean> {
    const request = encodeExistsRequest(key);
    const result = await this.sendRequest(request);
    return result && result.length > 0 && result[0] === 1;
  }

  async incr(key: string): Promise<bigint> {
    const request = encodeIncrRequest(key);
    const result = await this.sendRequest(request);
    return result.readBigUInt64BE(0);
  }

  async decr(key: string): Promise<bigint> {
    const request = encodeDecrRequest(key);
    const result = await this.sendRequest(request);
    return result.readBigUInt64BE(0);
  }

  async mset(keys: string[], values: Buffer[]): Promise<void> {
    const request = encodeMSetRequest(keys, values);
    await this.sendRequest(request);
  }

  async mget(keys: string[]): Promise<Array<Buffer | null>> {
    const request = encodeMGetRequest(keys);
    return await this.sendRequest(request);
  }

  async mdelete(keys: string[]): Promise<void> {
    const request = encodeMDeleteRequest(keys);
    await this.sendRequest(request);
  }

  close(): void {
    this.socket.destroy();
  }

  isConnected(): boolean {
    return !this.socket.destroyed && this.socket.readable && this.socket.writable;
  }
}

class ConnectionPool {
  private options: Required<FlinOptions>;
  private available: Connection[] = [];
  private activeCount = 0;
  private waiting: Array<{
    resolve: (conn: Connection) => void;
    reject: (err: Error) => void;
  }> = [];
  private closed = false;

  constructor(options: Required<FlinOptions>) {
    this.options = options;
    this.initialize();
  }

  private async initialize(): Promise<void> {
    const promises: Promise<void>[] = [];
    for (let i = 0; i < this.options.minConnections; i++) {
      promises.push(this.createConnection());
    }
    await Promise.all(promises);
  }

  private async createConnection(): Promise<void> {
    const conn = new Connection(this.options);
    await conn.connect();
    this.available.push(conn);
    this.activeCount++;
  }

  async acquire(): Promise<Connection> {
    if (this.closed) {
      throw new Error('Pool is closed');
    }

    if (this.available.length > 0) {
      const conn = this.available.pop()!;
      if (conn.isConnected()) {
        return conn;
      }
      this.activeCount--;
    }

    if (this.activeCount < this.options.maxConnections) {
      const conn = new Connection(this.options);
      await conn.connect();
      this.activeCount++;
      return conn;
    }

    return new Promise((resolve, reject) => {
      this.waiting.push({ resolve, reject });
    });
  }

  release(conn: Connection): void {
    if (this.closed) {
      conn.close();
      return;
    }

    const waiter = this.waiting.shift();
    if (waiter) {
      waiter.resolve(conn);
      return;
    }

    if (conn.isConnected()) {
      this.available.push(conn);
    } else {
      this.activeCount--;
    }
  }

  async close(): Promise<void> {
    this.closed = true;
    
    while (this.waiting.length > 0) {
      const waiter = this.waiting.shift();
      waiter?.reject(new Error('Pool closed'));
    }

    for (const conn of this.available) {
      conn.close();
    }
    this.available = [];
    this.activeCount = 0;
  }
}

// Protocol encoding functions
function encodeSetRequest(key: string, value: Buffer): Buffer {
  const keyBuf = Buffer.from(key);
  const keyLen = keyBuf.length;
  const valueLen = value.length;
  const totalSize = 1 + 4 + 2 + keyLen + 4 + valueLen;
  const buf = Buffer.allocUnsafe(totalSize);
  
  let pos = 0;
  buf[pos++] = OpCode.SET;
  
  const payloadLen = 2 + keyLen + 4 + valueLen;
  buf.writeUInt32BE(payloadLen, pos);
  pos += 4;
  
  buf.writeUInt16BE(keyLen, pos);
  pos += 2;
  keyBuf.copy(buf, pos);
  pos += keyLen;
  
  buf.writeUInt32BE(valueLen, pos);
  pos += 4;
  value.copy(buf, pos);
  
  return buf;
}

function encodeGetRequest(key: string): Buffer {
  return encodeSimpleRequest(OpCode.GET, key);
}

function encodeDeleteRequest(key: string): Buffer {
  return encodeSimpleRequest(OpCode.DEL, key);
}

function encodeExistsRequest(key: string): Buffer {
  return encodeSimpleRequest(OpCode.EXISTS, key);
}

function encodeIncrRequest(key: string): Buffer {
  return encodeSimpleRequest(OpCode.INCR, key);
}

function encodeDecrRequest(key: string): Buffer {
  return encodeSimpleRequest(OpCode.DECR, key);
}

function encodeSimpleRequest(opCode: number, key: string): Buffer {
  const keyBuf = Buffer.from(key);
  const keyLen = keyBuf.length;
  const totalSize = 1 + 4 + 2 + keyLen;
  const buf = Buffer.allocUnsafe(totalSize);
  
  let pos = 0;
  buf[pos++] = opCode;
  
  const payloadLen = 2 + keyLen;
  buf.writeUInt32BE(payloadLen, pos);
  pos += 4;
  
  buf.writeUInt16BE(keyLen, pos);
  pos += 2;
  keyBuf.copy(buf, pos);
  
  return buf;
}

function encodeMSetRequest(keys: string[], values: Buffer[]): Buffer {
  let totalSize = 1 + 4 + 2;
  const keyBuffers = keys.map(k => Buffer.from(k));
  
  for (let i = 0; i < keys.length; i++) {
    totalSize += 2 + keyBuffers[i].length + 4 + values[i].length;
  }
  
  const buf = Buffer.allocUnsafe(totalSize);
  let pos = 0;
  
  buf[pos++] = OpCode.MSET;
  
  const payloadLen = totalSize - 5;
  buf.writeUInt32BE(payloadLen, pos);
  pos += 4;
  
  buf.writeUInt16BE(keys.length, pos);
  pos += 2;
  
  for (let i = 0; i < keys.length; i++) {
    const keyLen = keyBuffers[i].length;
    buf.writeUInt16BE(keyLen, pos);
    pos += 2;
    keyBuffers[i].copy(buf, pos);
    pos += keyLen;
    
    const valueLen = values[i].length;
    buf.writeUInt32BE(valueLen, pos);
    pos += 4;
    values[i].copy(buf, pos);
    pos += valueLen;
  }
  
  return buf;
}

function encodeMGetRequest(keys: string[]): Buffer {
  return encodeBatchRequest(OpCode.MGET, keys);
}

function encodeMDeleteRequest(keys: string[]): Buffer {
  return encodeBatchRequest(OpCode.MDEL, keys);
}

function encodeBatchRequest(opCode: number, keys: string[]): Buffer {
  let totalSize = 1 + 4 + 2;
  const keyBuffers = keys.map(k => Buffer.from(k));
  
  for (const keyBuf of keyBuffers) {
    totalSize += 2 + keyBuf.length;
  }
  
  const buf = Buffer.allocUnsafe(totalSize);
  let pos = 0;
  
  buf[pos++] = opCode;
  
  const payloadLen = totalSize - 5;
  buf.writeUInt32BE(payloadLen, pos);
  pos += 4;
  
  buf.writeUInt16BE(keys.length, pos);
  pos += 2;
  
  for (const keyBuf of keyBuffers) {
    const keyLen = keyBuf.length;
    buf.writeUInt16BE(keyLen, pos);
    pos += 2;
    keyBuf.copy(buf, pos);
    pos += keyLen;
  }
  
  return buf;
}

export default FlinClient;
