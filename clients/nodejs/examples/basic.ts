import { FlinClient } from '../src/index';

async function main() {
  console.log('🚀 Flin Node.js SDK - Basic Example');
  console.log('====================================');

  // Create client
  const client = new FlinClient({
    host: 'localhost',
    port: 7380,
  });

  try {
    // Set a value
    console.log('\n📝 Setting key "greeting"...');
    await client.set('greeting', 'Hello, Flin!');
    console.log('✅ Set successful');

    // Get a value
    console.log('\n📖 Getting key "greeting"...');
    const value = await client.getString('greeting');
    console.log(`✅ Value: ${value}`);

    // Check if key exists
    console.log('\n🔍 Checking if key exists...');
    const exists = await client.exists('greeting');
    console.log(`✅ Exists: ${exists}`);

    // Counter operations
    console.log('\n🔢 Counter operations...');
    
    // Initialize counter
    await client.set('counter', Buffer.alloc(8));
    
    // Increment
    let count = await client.incr('counter');
    console.log(`✅ After increment: ${count}`);

    // Increment again
    count = await client.incr('counter');
    console.log(`✅ After second increment: ${count}`);

    // Decrement
    count = await client.decr('counter');
    console.log(`✅ After decrement: ${count}`);

    // Batch operations
    console.log('\n📦 Batch operations...');
    
    await client.mset([
      ['user:1', 'Alice'],
      ['user:2', 'Bob'],
      ['user:3', 'Charlie'],
    ]);
    console.log('✅ Batch set successful');

    const keys = ['user:1', 'user:2', 'user:3'];
    const results = await client.mget(keys);
    console.log('✅ Batch get results:');
    for (let i = 0; i < results.length; i++) {
      if (results[i]) {
        console.log(`   ${keys[i]}: ${results[i]!.toString()}`);
      }
    }

    // Delete
    console.log('\n🗑️  Deleting keys...');
    await client.mdelete(keys);
    console.log('✅ Batch delete successful');

    // Clean up
    await client.delete('greeting');
    await client.delete('counter');

    console.log('\n✨ Example completed!');
  } catch (err) {
    console.error('❌ Error:', err);
  } finally {
    await client.close();
  }
}

main();
