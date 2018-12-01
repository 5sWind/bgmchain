/*
  This file is part of bgmash.

  bgmash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  bgmash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with bgmash.  If not, see <http://www.gnu.org/licenses/>.
*/

/** @file bgmash.h
* @date 2015
*/
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define BGMASH_REVISION 23
#define BGMASH_DATASET_BYTES_INIT 1073741824U // 2**30
#define BGMASH_DATASET_BYTES_GROWTH 8388608U  // 2**23
#define BGMASH_CACHE_BYTES_INIT 1073741824U // 2**24
#define BGMASH_CACHE_BYTES_GROWTH 131072U  // 2**17
#define BGMASH_EPOCH_LENGTH 30000U
#define BGMASH_MIX_BYTES 128
#define BGMASH_HASH_BYTES 64
#define BGMASH_DATASET_PARENTS 256
#define BGMASH_CACHE_ROUNDS 3
#define BGMASH_ACCESSES 64
#define BGMASH_DAG_MAGIC_NUM_SIZE 8
#define BGMASH_DAG_MAGIC_NUM 0xFEE1DEADBADDCAFE

#ifdef __cplusplus
extern "C" {
#endif

/// Type of a seedhash/blockhash e.t.c.
typedef struct bgmash_h256 { uint8_t b[32]; } bgmash_h256_t;

// convenience macro to statically initialize an h256_t
// usage:
// bgmash_h256_t a = bgmash_h256_static_init(1, 2, 3, ... )
// have to provide all 32 values. If you don't provide all the rest
// will simply be unitialized (not guranteed to be 0)
#define bgmash_h256_static_init(...)			\
	{ {__VA_ARGS__} }

struct bgmash_light;
typedef struct bgmash_light* bgmash_light_t;
struct bgmash_full;
typedef struct bgmash_full* bgmash_full_t;
typedef int(*bgmash_callback_t)(unsigned);

typedef struct bgmash_return_value {
	bgmash_h256_t result;
	bgmash_h256_t mix_hash;
	bool success;
} bgmash_return_value_t;

/**
 * Allocate and initialize a new bgmash_light handler
 *
 * @param block_number   The block number for which to create the handler
 * @return               Newly allocated bgmash_light handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref bgmash_compute_cache_nodes()
 */
bgmash_light_t bgmash_light_new(uint64_t block_number);
/**
 * Frees a previously allocated bgmash_light handler
 * @param light        The light handler to free
 */
void bgmash_light_delete(bgmash_light_t light);
/**
 * Calculate the light client data
 *
 * @param light          The light client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               an object of bgmash_return_value_t holding the return values
 */
bgmash_return_value_t bgmash_light_compute(
	bgmash_light_t light,
	bgmash_h256_t const header_hash,
	uint64_t nonce
);

/**
 * Allocate and initialize a new bgmash_full handler
 *
 * @param light         The light handler containing the cache.
 * @param callback      A callback function with signature of @ref bgmash_callback_t
 *                      It accepts an unsigned with which a progress of DAG calculation
 *                      can be displayed. If all goes well the callback should return 0.
 *                      If a non-zero value is returned then DAG generation will stop.
 *                      Be advised. A progress value of 100 means that DAG creation is
 *                      almost complete and that this function will soon return succesfully.
 *                      It does not mean that the function has already had a succesfull return.
 * @return              Newly allocated bgmash_full handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref bgmash_compute_full_data()
 */
bgmash_full_t bgmash_full_new(bgmash_light_t light, bgmash_callback_t callback);

/**
 * Frees a previously allocated bgmash_full handler
 * @param full    The light handler to free
 */
void bgmash_full_delete(bgmash_full_t full);
/**
 * Calculate the full client data
 *
 * @param full           The full client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               An object of bgmash_return_value to hold the return value
 */
bgmash_return_value_t bgmash_full_compute(
	bgmash_full_t full,
	bgmash_h256_t const header_hash,
	uint64_t nonce
);
/**
 * Get a pointer to the full DAG data
 */
void const* bgmash_full_dag(bgmash_full_t full);
/**
 * Get the size of the DAG data
 */
uint64_t bgmash_full_dag_size(bgmash_full_t full);

/**
 * Calculate the seedhash for a given block number
 */
bgmash_h256_t bgmash_get_seedhash(uint64_t block_number);

#ifdef __cplusplus
}
#endif
