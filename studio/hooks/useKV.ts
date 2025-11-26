'use client';

import axios from 'axios';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

// Configure API base URL - point to the Flin HTTP API server
const API_BASE_URL = process.env.NEXT_PUBLIC_FLIN_API_URL || 'http://localhost:8888';

// Create axios instance
const axiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
});

// Types
export interface KVItem {
  key: string;
  value: string;
  size?: number;
}

export interface KVListResponse {
  items: KVItem[];
  total: number;
}

export interface SetKVRequest {
  key: string;
  value: string;
  ttl?: number;
}

export interface UpdateKVRequest {
  key: string;
  value: string;
  ttl?: number;
}

// Query Keys
export const kvQueryKeys = {
  all: ['kv'] as const,
  keys: () => [...kvQueryKeys.all, 'keys'] as const,
  key: (key: string) => [...kvQueryKeys.all, 'key', key] as const,
};

// Queries

/**
 * Hook to fetch all KV keys
 * GET /kv/keys
 */
export function useKVKeys(enabled = true) {
  return useQuery({
    queryKey: kvQueryKeys.keys(),
    queryFn: async () => {
      const response = await axiosInstance.get<KVListResponse>('/kv/keys');
      return response.data;
    },
    enabled,
    staleTime: 30000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes (formerly cacheTime)
  });
}

/**
 * Hook to fetch a specific KV value
 * GET /kv/get?key=...
 */
export function useKVGet(key: string | null) {
  return useQuery({
    queryKey: kvQueryKeys.key(key || ''),
    queryFn: async () => {
      const response = await axiosInstance.get<KVItem>('/kv/get', {
        params: { key },
      });
      return response.data;
    },
    enabled: !!key,
    staleTime: 30000,
    gcTime: 5 * 60 * 1000,
  });
}

// Mutations

/**
 * Hook to set a KV value
 * POST /kv/set
 */
export function useSetKV() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: SetKVRequest) => {
      const response = await axiosInstance.post('/kv/set', request);
      return response.data;
    },
    onSuccess: () => {
      // Invalidate KV keys list to refetch
      queryClient.invalidateQueries({ queryKey: kvQueryKeys.keys() });
    },
    onError: (error: any) => {
      console.error('Failed to set KV:', error.response?.data || error.message);
    },
  });
}

/**
 * Hook to update a KV value
 * POST /kv/update
 */
export function useUpdateKV() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: UpdateKVRequest) => {
      const response = await axiosInstance.post('/kv/update', request);
      return response.data;
    },
    onSuccess: (_, variables) => {
      // Invalidate specific key and keys list
      queryClient.invalidateQueries({ queryKey: kvQueryKeys.key(variables.key) });
      queryClient.invalidateQueries({ queryKey: kvQueryKeys.keys() });
    },
    onError: (error: any) => {
      console.error('Failed to update KV:', error.response?.data || error.message);
    },
  });
}

/**
 * Hook to delete a KV value
 * POST /kv/delete
 */
export function useDeleteKV() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (key: string) => {
      const response = await axiosInstance.post('/kv/delete', { key });
      return response.data;
    },
    onSuccess: (_, key) => {
      // Invalidate specific key and keys list
      queryClient.invalidateQueries({ queryKey: kvQueryKeys.key(key) });
      queryClient.invalidateQueries({ queryKey: kvQueryKeys.keys() });
    },
    onError: (error: any) => {
      console.error('Failed to delete KV:', error.response?.data || error.message);
    },
  });
}

export default {
  useKVKeys,
  useKVGet,
  useSetKV,
  useUpdateKV,
  useDeleteKV,
};
