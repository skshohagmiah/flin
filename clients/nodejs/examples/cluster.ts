import { ClusterClient } from '../src/index';

async function main() {
  console.log('🚀 Flin Node.js SDK - Cluster Example');
  console.log('======================================');

  // Create cluster-aware client
  const client = new ClusterClient({
    httpAddresses: [
      'localhost:7080',
      'localhost:7081',
      'localhost:7082',
    ],
    minConnectionsPerNode: 2,
    maxConnectionsPerNode: 10,
    refreshInterval: 30000,
  });

  try {
    // Initialize and discover topology
    await client.initialize();
    console.log('✅ Connected to cluster');

    // Get topology info
    const topology = client.getTopology();
    console.log('\n📊 Cluster Topology:');
    console.log(`   Nodes: ${topology.nodes.length}`);
    console.log(`   Partitions: ${topology.partitionMap.size}`);
    console.log(`   Last Update: ${topology.lastUpdate}`);

    // Set values (automatically routed to correct nodes)
    console.log('\n📝 Setting values across cluster...');
    
    await client.set('user:1', 'Alice');
    await client.set('user:2', 'Bob');
    await client.set('user:3', 'Charlie');
    
    console.log('✅ Values set across cluster');

    // Get values (automatically routed)
    console.log('\n📖 Getting values from cluster...');
    
    const value = await client.getString('user:1');
    console.log(`   user:1 = ${value}`);

    // Batch operations (distributed across nodes)
    console.log('\n📦 Batch operations across cluster...');
    
    await client.mset([
      ['key1', 'value1'],
      ['key2', 'value2'],
      ['key3', 'value3'],
      ['key4', 'value4'],
      ['key5', 'value5'],
    ]);
    console.log('✅ Batch set distributed across nodes');

    const keys = ['key1', 'key2', 'key3', 'key4', 'key5'];
    const results = await client.mget(keys);
    console.log('✅ Batch get from multiple nodes:');
    for (let i = 0; i < results.length; i++) {
      if (results[i]) {
        console.log(`   ${keys[i]}: ${results[i]!.toString()}`);
      }
    }

    // Counter across cluster
    console.log('\n🔢 Counter operations...');
    
    // Initialize counter
    await client.set('global:counter', Buffer.alloc(8));
    
    const count = await client.incr('global:counter');
    console.log(`✅ Counter incremented: ${count}`);

    // Clean up
    console.log('\n🗑️  Cleaning up...');
    await client.delete('user:1');
    await client.delete('user:2');
    await client.delete('user:3');
    await client.mdelete(keys);
    await client.delete('global:counter');

    console.log('\n✨ Cluster example completed!');
  } catch (err) {
    console.error('❌ Error:', err);
  } finally {
    await client.close();
  }
}

main();
