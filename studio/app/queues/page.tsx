"use client";

import { Modal } from "@/components/modal";
import { Pagination } from "@/components/pagination";
import { useState } from "react";
import { useQueues, useCreateQueue, usePushQueue, usePopQueue, useDeleteQueue } from "@/hooks/useQueues";

interface Queue {
  id: string;
  name: string;
  depth: string;
  inRate: string;
  outRate: string;
  status: string;
}

export default function QueuesPage() {
  const [currentPage, setCurrentPage] = useState(1);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isPushModalOpen, setIsPushModalOpen] = useState(false);
  const [selectedQueue, setSelectedQueue] = useState<Queue | null>(null);
  const [queueName, setQueueName] = useState("");
  const [queueMessage, setQueueMessage] = useState("");
  const [queueDescription, setQueueDescription] = useState("");

  // React Query hooks
  const { data: queuesData, isLoading: queuesLoading, error: queuesError } = useQueues();
  const { mutate: createQueue } = useCreateQueue();
  const { mutate: pushQueue } = usePushQueue();
  const { mutate: popQueue } = usePopQueue();
  const { mutate: deleteQueue } = useDeleteQueue();

  const itemsPerPage = 5;
  
  // Transform API data to table format
  const allQueues: Queue[] = (queuesData?.items || []).map((item: any, index: number) => ({
    id: String(index + 1),
    name: item.name,
    depth: String(item.depth || 0),
    inRate: "—",
    outRate: "—",
    status: item.depth > 0 ? "active" : "idle",
  }));

  const totalPages = Math.ceil(allQueues.length / itemsPerPage);
  const paginatedQueues = allQueues.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  );

  const handleAddQueue = () => {
    if (queueName) {
      createQueue(queueName, {
        onSuccess: () => {
          setQueueName("");
          setQueueDescription("");
          setIsAddModalOpen(false);
        }
      });
    }
  };

  const handleEditQueue = (queue: Queue) => {
    setSelectedQueue(queue);
    setQueueName(queue.name);
    setIsEditModalOpen(true);
  };

  const handleSaveEdit = () => {
    if (selectedQueue && queueName) {
      // Queue names typically can't be edited, but you could implement renaming here
      console.log("Edit queue:", { ...selectedQueue, name: queueName });
      setIsEditModalOpen(false);
      setSelectedQueue(null);
    }
  };

  const handleDeleteQueue = (queue: Queue) => {
    setSelectedQueue(queue);
    setIsDeleteModalOpen(true);
  };

  const confirmDelete = () => {
    if (selectedQueue) {
      deleteQueue(selectedQueue.name, {
        onSuccess: () => {
          setIsDeleteModalOpen(false);
          setSelectedQueue(null);
        }
      });
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Queues</h1>
        <p className="text-sm text-muted-foreground">
          Inspect queue depth, throughput and recent messages. All controls are ready for when Flin queue APIs are wired up.
        </p>
      </div>

      {/* Main Content Card */}
      <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm overflow-hidden">
        {/* Toolbar */}
        <div className="px-6 py-4 border-b border-border/50 bg-gradient-to-b from-muted/30 to-transparent">
          <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
            {/* Search Input */}
            <div className="flex-1 md:max-w-sm">
              <label className="block text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                Filter queues
              </label>
              <div className="relative">
                <svg
                  className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <input
                  placeholder="tasks:* or jobs:*"
                  className="w-full rounded-lg border border-border/50 bg-background/50 pl-10 pr-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:bg-background focus:ring-2 focus:ring-blue-500/20 placeholder:text-muted-foreground/50"
                />
              </div>
            </div>

            {/* Action Buttons */}
            <div className="flex gap-2 w-full md:w-auto">
              <button className="flex-1 md:flex-none rounded-lg border border-border/50 px-4 py-2 text-sm font-medium text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-all hover:border-border">
                <span className="flex items-center justify-center gap-2">
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  Refresh
                </span>
              </button>
              <button
                onClick={() => setIsAddModalOpen(true)}
                className="flex-1 md:flex-none rounded-lg bg-blue-600 hover:bg-blue-700 px-4 py-2 text-sm font-medium text-white transition-all shadow-sm hover:shadow-md dark:bg-blue-600 dark:hover:bg-blue-500"
              >
                <span className="flex items-center justify-center gap-2">
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  New queue
                </span>
              </button>
            </div>
          </div>
        </div>

        {/* Table */}
        <div className="overflow-x-auto">
          {queuesLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
                <p className="text-sm text-muted-foreground">Loading queues...</p>
              </div>
            </div>
          ) : queuesError ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <p className="text-sm text-red-600 dark:text-red-400 mb-2">Error loading queues</p>
                <p className="text-xs text-muted-foreground">{(queuesError as any).message || 'Failed to fetch from API'}</p>
              </div>
            </div>
          ) : allQueues.length === 0 ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <svg className="w-12 h-12 text-muted-foreground/30 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 7l-8-4-8 4m0 0l8 4m-8-4v10l8 4m0-10l8 4m-8-4l8-4" />
                </svg>
                <p className="text-sm text-muted-foreground">No queues found</p>
                <p className="text-xs text-muted-foreground/70 mt-1">Create a queue to get started</p>
              </div>
            </div>
          ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border/50 bg-muted/20">
                <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Queue Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Depth
                </th>
                <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  In/s
                </th>
                <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Out/s
                </th>
                <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-right text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/30">
              {paginatedQueues.map((q, idx) => (
                <tr
                  key={idx}
                  className="hover:bg-muted/20 transition-colors group"
                >
                  <td className="px-6 py-4 font-mono text-xs text-blue-700 dark:text-blue-400 font-semibold">
                    {q.name}
                  </td>
                  <td className="px-6 py-4 text-sm font-medium">
                    {q.depth}
                  </td>
                  <td className="px-6 py-4 text-sm text-muted-foreground">
                    {q.inRate}
                  </td>
                  <td className="px-6 py-4 text-sm text-muted-foreground">
                    {q.outRate}
                  </td>
                  <td className="px-6 py-4 text-sm">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-500/10 text-gray-700 dark:text-gray-400 border border-gray-500/20">
                      {q.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleEditQueue(q)}
                        className="p-2 text-white dark:text-blue-300 bg-blue-600 dark:bg-blue-950 hover:bg-blue-700 dark:hover:bg-blue-900 rounded-lg transition-all border border-blue-700 dark:border-blue-800"
                        title="Edit"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                      </button>
                      <button
                        onClick={() => handleDeleteQueue(q)}
                        className="p-2 text-white dark:text-red-300 bg-red-600 dark:bg-red-950 hover:bg-red-700 dark:hover:bg-red-900 rounded-lg transition-all border border-red-700 dark:border-red-800"
                        title="Delete"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          )}
        </div>

        {/* Pagination */}
        <Pagination
          currentPage={currentPage}
          totalPages={totalPages}
          onPageChange={setCurrentPage}
          itemsPerPage={itemsPerPage}
          totalItems={allQueues.length}
        />
      </div>

      {/* Add Modal */}
      <Modal
        isOpen={isAddModalOpen}
        title="Create New Queue"
        description="Set up a new message queue"
        onClose={() => setIsAddModalOpen(false)}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              Queue Name
            </label>
            <input
              type="text"
              placeholder="e.g., tasks:email"
              value={queueName}
              onChange={(e) => setQueueName(e.target.value)}
              className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              Description (optional)
            </label>
            <textarea
              placeholder="Describe what this queue is used for..."
              value={queueDescription}
              onChange={(e) => setQueueDescription(e.target.value)}
              rows={3}
              className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20 resize-none"
            />
          </div>

          <div className="flex gap-3 justify-end pt-4 border-t border-border/50">
            <button
              onClick={() => setIsAddModalOpen(false)}
              className="px-4 py-2 text-sm font-medium text-muted-foreground rounded-lg border border-border/50 hover:bg-muted/50 transition-all"
            >
              Cancel
            </button>
            <button
              onClick={handleAddQueue}
              className="px-4 py-2 text-sm font-medium text-white rounded-lg bg-blue-600 hover:bg-blue-700 transition-all"
            >
              Create Queue
            </button>
          </div>
        </div>
      </Modal>

      {/* Edit Modal */}
      <Modal
        isOpen={isEditModalOpen}
        title="Edit Queue"
        description={`Editing: ${selectedQueue?.name}`}
        onClose={() => setIsEditModalOpen(false)}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              Queue Name
            </label>
            <input
              type="text"
              disabled
              value={queueName}
              className="w-full rounded-lg border border-border/50 bg-muted/50 px-3 py-2 text-sm outline-none opacity-60 cursor-not-allowed"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              Description
            </label>
            <textarea
              placeholder="Queue description..."
              value={queueDescription}
              onChange={(e) => setQueueDescription(e.target.value)}
              rows={3}
              className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20 resize-none"
            />
          </div>

          <div className="flex gap-3 justify-end pt-4 border-t border-border/50">
            <button
              onClick={() => setIsEditModalOpen(false)}
              className="px-4 py-2 text-sm font-medium text-muted-foreground rounded-lg border border-border/50 hover:bg-muted/50 transition-all"
            >
              Cancel
            </button>
            <button
              onClick={handleSaveEdit}
              className="px-4 py-2 text-sm font-medium text-white rounded-lg bg-blue-600 hover:bg-blue-700 transition-all"
            >
              Save Changes
            </button>
          </div>
        </div>
      </Modal>

      {/* Delete Modal */}
      <Modal
        isOpen={isDeleteModalOpen}
        title="Delete Queue"
        description="Are you sure? This action cannot be undone."
        onClose={() => setIsDeleteModalOpen(false)}
        size="sm"
      >
        <div className="space-y-4">
          <div className="p-4 rounded-lg bg-red-500/10 border border-red-500/20">
            <p className="text-sm text-red-700 dark:text-red-400">
              You are about to delete <strong className="font-mono">{selectedQueue?.name}</strong>
            </p>
          </div>

          <div className="flex gap-3 justify-end pt-4 border-t border-border/50">
            <button
              onClick={() => setIsDeleteModalOpen(false)}
              className="px-4 py-2 text-sm font-medium text-muted-foreground rounded-lg border border-border/50 hover:bg-muted/50 transition-all"
            >
              Cancel
            </button>
            <button
              onClick={confirmDelete}
              className="px-4 py-2 text-sm font-medium text-white rounded-lg bg-red-600 hover:bg-red-700 transition-all"
            >
              Delete
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
