"use client";

import { Modal } from "@/components/modal";
import { Pagination } from "@/components/pagination";
import { useState } from "react";
import { useKVKeys, useSetKV, useUpdateKV, useDeleteKV, useKVGet } from "@/hooks/useKV";

interface KVItem {
  id: string;
  key: string;
  value: string;
  type: string;
  size: string;
  ttl: string;
  status: string;
}

export default function KvPage() {
  const [currentPage, setCurrentPage] = useState(1);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isViewModalOpen, setIsViewModalOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState<KVItem | null>(null);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [newType, setNewType] = useState("string");
  const [newTTL, setNewTTL] = useState("");

  // React Query hooks
  const { data: keysData, isLoading: keysLoading, error: keysError } = useKVKeys();
  const { mutate: setKV } = useSetKV();
  const { mutate: updateKV } = useUpdateKV();
  const { mutate: deleteKV } = useDeleteKV();
  const { data: valueData } = useKVGet(selectedItem?.key || "");

  const itemsPerPage = 5;
  
  // Transform API data to table format
  const allKeys: KVItem[] = (keysData?.items || []).map((item: any, index: number) => ({
    id: String(index + 1),
    key: item.key,
    value: item.value || "",
    type: "string",
    size: item.size ? `${item.size} B` : "—",
    ttl: "—",
    status: "active",
  }));

  const totalPages = Math.ceil(allKeys.length / itemsPerPage);
  const paginatedKeys = allKeys.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  );

  const handleAddKey = () => {
    if (newKey && newValue) {
      setKV({ key: newKey, value: newValue }, {
        onSuccess: () => {
          setNewKey("");
          setNewValue("");
          setNewType("string");
          setNewTTL("");
          setIsAddModalOpen(false);
        }
      });
    }
  };

  const handleEditKey = (item: KVItem) => {
    setSelectedItem(item);
    setNewKey(item.key);
    setNewValue(item.value);
    setNewType(item.type);
    setNewTTL(item.ttl === "—" ? "" : item.ttl);
    setIsEditModalOpen(true);
  };

  const handleSaveEdit = () => {
    if (selectedItem && newKey && newValue) {
      updateKV({ key: newKey, value: newValue }, {
        onSuccess: () => {
          setIsEditModalOpen(false);
          setSelectedItem(null);
        }
      });
    }
  };

  const handleDeleteKey = (item: KVItem) => {
    setSelectedItem(item);
    setIsDeleteModalOpen(true);
  };

  const confirmDelete = () => {
    if (selectedItem) {
      deleteKV(selectedItem.key, {
        onSuccess: () => {
          setIsDeleteModalOpen(false);
          setSelectedItem(null);
        }
      });
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">KV Store</h1>
        <p className="text-sm text-muted-foreground">
          Browse keys, inspect values and run read/write operations once connected to a Flin node.
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
                Filter keys
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
                  placeholder="user:* or orders:2024-*"
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
                  Scan keys
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
                  New key
                </span>
              </button>
            </div>
          </div>
        </div>

        {/* Table */}
        <div className="overflow-x-auto">
          {keysLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
                <p className="text-sm text-muted-foreground">Loading KV keys...</p>
              </div>
            </div>
          ) : keysError ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <p className="text-sm text-red-600 dark:text-red-400 mb-2">Error loading KV keys</p>
                <p className="text-xs text-muted-foreground">{(keysError as any).message || 'Failed to fetch from API'}</p>
              </div>
            </div>
          ) : allKeys.length === 0 ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <svg className="w-12 h-12 text-muted-foreground/30 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                </svg>
                <p className="text-sm text-muted-foreground">No keys found</p>
                <p className="text-xs text-muted-foreground/70 mt-1">Add a key to get started</p>
              </div>
            </div>
          ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border/50 bg-muted/20">
                <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Key
                </th>
                <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Size
                </th>
                <th className="px-6 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  TTL
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
              {paginatedKeys.map((row, idx) => (
                <tr
                  key={idx}
                  className="hover:bg-muted/20 transition-colors group"
                >
                  <td className="px-6 py-4 font-mono text-xs text-blue-700 dark:text-blue-400 font-semibold">
                    {row.key}
                  </td>
                  <td className="px-6 py-4 text-sm">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-500/10 text-purple-700 dark:text-purple-400 border border-purple-500/20">
                      {row.type}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-muted-foreground">
                    {row.size}
                  </td>
                  <td className="px-6 py-4 text-sm text-muted-foreground">
                    {row.ttl}
                  </td>
                  <td className="px-6 py-4 text-sm">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-500/20">
                      <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 mr-1.5" />
                      {row.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleEditKey(row)}
                        className="p-2 text-white dark:text-blue-300 bg-blue-600 dark:bg-blue-950 hover:bg-blue-700 dark:hover:bg-blue-900 rounded-lg transition-all border border-blue-700 dark:border-blue-800"
                        title="Edit"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                      </button>
                      <button
                        onClick={() => handleDeleteKey(row)}
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
          totalItems={allKeys.length}
        />
      </div>

      {/* Add Modal */}
      <Modal
        isOpen={isAddModalOpen}
        title="Add New Key"
        description="Create a new key-value pair in your KV store"
        onClose={() => setIsAddModalOpen(false)}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              Key Name
            </label>
            <input
              type="text"
              placeholder="e.g., user:123"
              value={newKey}
              onChange={(e) => setNewKey(e.target.value)}
              className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              Value
            </label>
            <textarea
              placeholder="Enter your value..."
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              rows={3}
              className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20 resize-none"
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">
                Type
              </label>
              <select
                value={newType}
                onChange={(e) => setNewType(e.target.value)}
                className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20"
              >
                <option>string</option>
                <option>json</option>
                <option>list</option>
                <option>hash</option>
                <option>zset</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-1">
                TTL (optional)
              </label>
              <input
                type="text"
                placeholder="e.g., 30m, 1h"
                value={newTTL}
                onChange={(e) => setNewTTL(e.target.value)}
                className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20"
              />
            </div>
          </div>

          <div className="flex gap-3 justify-end pt-4 border-t border-border/50">
            <button
              onClick={() => setIsAddModalOpen(false)}
              className="px-4 py-2 text-sm font-medium text-muted-foreground rounded-lg border border-border/50 hover:bg-muted/50 transition-all"
            >
              Cancel
            </button>
            <button
              onClick={handleAddKey}
              className="px-4 py-2 text-sm font-medium text-white rounded-lg bg-blue-600 hover:bg-blue-700 transition-all"
            >
              Add Key
            </button>
          </div>
        </div>
      </Modal>

      {/* Edit Modal */}
      <Modal
        isOpen={isEditModalOpen}
        title="Edit Key"
        description={`Editing: ${selectedItem?.key}`}
        onClose={() => setIsEditModalOpen(false)}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              Key Name
            </label>
            <input
              type="text"
              disabled
              value={newKey}
              className="w-full rounded-lg border border-border/50 bg-muted/50 px-3 py-2 text-sm outline-none opacity-60 cursor-not-allowed"
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">
                Type
              </label>
              <select
                value={newType}
                onChange={(e) => setNewType(e.target.value)}
                className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20"
              >
                <option>string</option>
                <option>json</option>
                <option>list</option>
                <option>hash</option>
                <option>zset</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-1">
                TTL (optional)
              </label>
              <input
                type="text"
                placeholder="e.g., 30m, 1h"
                value={newTTL}
                onChange={(e) => setNewTTL(e.target.value)}
                className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm outline-none transition-all focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20"
              />
            </div>
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
        title="Delete Key"
        description="Are you sure? This action cannot be undone."
        onClose={() => setIsDeleteModalOpen(false)}
        size="sm"
      >
        <div className="space-y-4">
          <div className="p-4 rounded-lg bg-red-500/10 border border-red-500/20">
            <p className="text-sm text-red-700 dark:text-red-400">
              You are about to delete <strong className="font-mono">{selectedItem?.key}</strong>
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
