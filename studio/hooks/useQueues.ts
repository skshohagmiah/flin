'use client';

import axios from 'axios';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

// Configure API base URL
const API_BASE_URL = process.env.NEXT_PUBLIC_FLIN_API_URL || 'http://localhost:8888';

const axiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
});

// Types
export interface Queue {
  name: string;
  depth: number;
}

export interface QueueListResponse {
  items: Queue[];
  total: number;
}

export interface PushRequest {
  queue: string;
  message: string;
}

export interface PopRequest {
  queue: string;
}

export interface PopResponse {
  message: string;
}

// Query Keys
export const queueQueryKeys = {
  all: ['queues'] as const,
  list: () => [...queueQueryKeys.all, 'list'] as const,
  queue: (name: string) => [...queueQueryKeys.all, name] as const,
};

// Queries

/**
 * Hook to fetch all queues
 * GET /queues
 */
export function useQueues(enabled = true) {
  return useQuery({
    queryKey: queueQueryKeys.list(),
    queryFn: async () => {
      const response = await axiosInstance.get<QueueListResponse>('/queues');
      return response.data;
    },
    enabled,
    staleTime: 30000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
  });
}

// Mutations

/**
 * Hook to push a message to a queue
 * POST /queues/push
 */
export function usePushQueue() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: PushRequest) => {
      const response = await axiosInstance.post('/queues/push', request);
      return response.data;
    },
    onSuccess: (_, variables) => {
      // Invalidate queue list
      queryClient.invalidateQueries({ queryKey: queueQueryKeys.list() });
      queryClient.invalidateQueries({ queryKey: queueQueryKeys.queue(variables.queue) });
    },
    onError: (error: any) => {
      console.error('Failed to push to queue:', error.response?.data || error.message);
    },
  });
}

/**
 * Hook to pop a message from a queue
 * POST /queues/pop
 */
export function usePopQueue() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (queueName: string) => {
      const response = await axiosInstance.post<PopResponse>('/queues/pop', {
        queue: queueName,
      });
      return response.data;
    },
    onSuccess: (_, queueName) => {
      // Invalidate queue list
      queryClient.invalidateQueries({ queryKey: queueQueryKeys.list() });
      queryClient.invalidateQueries({ queryKey: queueQueryKeys.queue(queueName) });
    },
    onError: (error: any) => {
      console.error('Failed to pop from queue:', error.response?.data || error.message);
    },
  });
}

/**
 * Hook to create a new queue
 * POST /queues/create
 */
export function useCreateQueue() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (queueName: string) => {
      const response = await axiosInstance.post('/queues/create', {
        name: queueName,
      });
      return response.data;
    },
    onSuccess: () => {
      // Invalidate queue list
      queryClient.invalidateQueries({ queryKey: queueQueryKeys.list() });
    },
    onError: (error: any) => {
      console.error('Failed to create queue:', error.response?.data || error.message);
    },
  });
}

/**
 * Hook to delete a queue
 * POST /queues/delete
 */
export function useDeleteQueue() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (queueName: string) => {
      const response = await axiosInstance.post('/queues/delete', {
        name: queueName,
      });
      return response.data;
    },
    onSuccess: (_, queueName) => {
      // Invalidate queue list and specific queue
      queryClient.invalidateQueries({ queryKey: queueQueryKeys.list() });
      queryClient.invalidateQueries({ queryKey: queueQueryKeys.queue(queueName) });
    },
    onError: (error: any) => {
      console.error('Failed to delete queue:', error.response?.data || error.message);
    },
  });
}

export default {
  useQueues,
  usePushQueue,
  usePopQueue,
  useCreateQueue,
  useDeleteQueue,
};
