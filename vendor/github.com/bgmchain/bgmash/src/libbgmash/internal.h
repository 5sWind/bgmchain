#pragma once
#include "compiler.h"
#include "endian.h"
#include "bgmash.h"
#include <stdio.h>

#define ENABLE_SSE 0

#if defined(_M_X64) && ENABLE_SSE
#include <smmintrin.h>
#endif

#ifdef __cplusplus
extern "C" {
#endif

// compile time settings
#define NODE_WORDS (64/4)
#define MIX_WORDS (BGMASH_MIX_BYTES/4)
#define MIX_NODES (MIX_WORDS / NODE_WORDS)
#include <stdint.h>

typedef union node {
	uint8_t bytes[NODE_WORDS * 4];
	uint32_t words[NODE_WORDS];
	uint64_t double_words[NODE_WORDS / 2];

#if defined(_M_X64) && ENABLE_SSE
	__m128i xmm[NODE_WORDS/4];
#endif

} node;

static inline uint8_t bgmash_h256_get(bgmash_h256_t const* hash, unsigned int i)
{
	return hash->b[i];
}

static inline void bgmash_h256_set(bgmash_h256_t* hash, unsigned int i, uint8_t v)
{
	hash->b[i] = v;
}

static inline void bgmash_h256_reset(bgmash_h256_t* hash)
{
	memset(hash, 0, 32);
}

// Returns if hash is less than or equal to boundary (2^256/difficulty)
static inline bool bgmash_check_difficulty(
	bgmash_h256_t const* hash,
	bgmash_h256_t const* boundary
)
{
	// Boundary is big endian
	for (int i = 0; i < 32; i++) {
		if (bgmash_h256_get(hash, i) == bgmash_h256_get(boundary, i)) {
			continue;
		}
		return bgmash_h256_get(hash, i) < bgmash_h256_get(boundary, i);
	}
	return true;
}

/**
 *  Difficulty quick check for POW preverification
 *
 * @param header_hash      The hash of the header
 * @param nonce            The block's nonce
 * @param mix_hash         The mix digest hash
 * @param boundary         The boundary is defined as (2^256 / difficulty)
 * @return                 true for succesful pre-verification and false otherwise
 */
bool bgmash_quick_check_difficulty(
	bgmash_h256_t const* header_hash,
	uint64_t const nonce,
	bgmash_h256_t const* mix_hash,
	bgmash_h256_t const* boundary
);

struct bgmash_light {
	void* cache;
	uint64_t cache_size;
	uint64_t block_number;
};

/**
 * Allocate and initialize a new bgmash_light handler. Internal version
 *
 * @param cache_size    The size of the cache in bytes
 * @param seed          Block seedhash to be used during the computation of the
 *                      cache nodes
 * @return              Newly allocated bgmash_light handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref bgmash_compute_cache_nodes()
 */
bgmash_light_t bgmash_light_new_internal(uint64_t cache_size, bgmash_h256_t const* seed);

/**
 * Calculate the light client data. Internal version.
 *
 * @param light          The light client handler
 * @param full_size      The size of the full data in bytes.
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               The resulting hash.
 */
bgmash_return_value_t bgmash_light_compute_internal(
	bgmash_light_t light,
	uint64_t full_size,
	bgmash_h256_t const header_hash,
	uint64_t nonce
);

struct bgmash_full {
	FILE* file;
	uint64_t file_size;
	node* data;
};

/**
 * Allocate and initialize a new bgmash_full handler. Internal version.
 *
 * @param dirname        The directory in which to put the DAG file.
 * @param seedhash       The seed hash of the block. Used in the DAG file naming.
 * @param full_size      The size of the full data in bytes.
 * @param cache          A cache object to use that was allocated with @ref bgmash_cache_new().
 *                       Iff this function succeeds the bgmash_full_t will take memory
 *                       memory ownership of the cache and free it at deletion. If
 *                       not then the user still has to handle freeing of the cache himself.
 * @param callback       A callback function with signature of @ref bgmash_callback_t
 *                       It accepts an unsigned with which a progress of DAG calculation
 *                       can be displayed. If all goes well the callback should return 0.
 *                       If a non-zero value is returned then DAG generation will stop.
 * @return               Newly allocated bgmash_full handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref bgmash_compute_full_data()
 */
bgmash_full_t bgmash_full_new_internal(
	char const* dirname,
	bgmash_h256_t const seed_hash,
	uint64_t full_size,
	bgmash_light_t const light,
	bgmash_callback_t callback
);

void bgmash_calculate_dag_item(
	node* const ret,
	uint32_t node_index,
	bgmash_light_t const cache
);

void bgmash_quick_hash(
	bgmash_h256_t* return_hash,
	bgmash_h256_t const* header_hash,
	const uint64_t nonce,
	bgmash_h256_t const* mix_hash
);

uint64_t bgmash_get_datasize(uint64_t const block_number);
uint64_t bgmash_get_cachesize(uint64_t const block_number);

/**
 * Compute the memory data for a full node's memory
 *
 * @param mem         A pointer to an bgmash full's memory
 * @param full_size   The size of the full data in bytes
 * @param cache       A cache object to use in the calculation
 * @param callback    The callback function. Check @ref bgmash_full_new() for details.
 * @return            true if all went fine and false for invalid parameters
 */
bool bgmash_compute_full_data(
	void* mem,
	uint64_t full_size,
	bgmash_light_t const light,
	bgmash_callback_t callback
);

#ifdef __cplusplus
}
#endif
