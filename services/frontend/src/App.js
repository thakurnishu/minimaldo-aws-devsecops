import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import './App.css';
import { API_URL } from './utils/env';

const EditTodoForm = ({ todo, onSave, onCancel }) => {
  const [title, setTitle] = useState(todo.title);
  const [description, setDescription] = useState(todo.description || '');

  const handleSubmit = (e) => {
    e.preventDefault();
    onSave({ ...todo, title, description });
  };

  return (
    <form onSubmit={handleSubmit} className="edit-form">
      <input
        type="text"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        className="edit-input"
        required
      />
      <textarea
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        className="edit-textarea"
        placeholder="Add description (optional)"
      />
      <div className="edit-actions">
        <button type="submit" className="save-btn">
          Save
        </button>
        <button type="button" onClick={onCancel} className="cancel-btn">
          Cancel
        </button>
      </div>
    </form>
  );
};

EditTodoForm.propTypes = {
  todo: PropTypes.shape({
    id: PropTypes.oneOfType([PropTypes.string, PropTypes.number]).isRequired,
    title: PropTypes.string.isRequired,
    description: PropTypes.string,
    completed: PropTypes.bool.isRequired,
    created_at: PropTypes.string,
    updated_at: PropTypes.string
  }).isRequired,
  onSave: PropTypes.func.isRequired,
  onCancel: PropTypes.func.isRequired
};

function App() {
  // State management
  const [todos, setTodos] = useState([]);
  const [dateGroups, setDateGroups] = useState([]);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [editingId, setEditingId] = useState(null);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [draggedItem, setDraggedItem] = useState(null);
  const [dragOverItem, setDragOverItem] = useState(null);
  const [dateRange, setDateRange] = useState('day');
  const [currentDate, setCurrentDate] = useState(new Date());

  // Date navigation functions
  const navigateDate = (direction) => {
    const newDate = new Date(currentDate);
    const oneYearAgo = new Date();
    oneYearAgo.setFullYear(oneYearAgo.getFullYear() - 1);

    switch (dateRange) {
      case 'day':
        newDate.setDate(newDate.getDate() + (direction === 'next' ? 1 : -1));
        break;
      case 'week':
        newDate.setDate(newDate.getDate() + (direction === 'next' ? 7 : -7));
        break;
      case 'month':
        newDate.setMonth(newDate.getMonth() + (direction === 'next' ? 1 : -1));
        break;
      default:
        break;
    }

    if (newDate >= oneYearAgo) {
      setCurrentDate(newDate);
    }
  };

  const handleRangeChange = (e) => {
    setDateRange(e.target.value);
  };

  const formatPeriodHeader = () => {
    switch (dateRange) {
      case 'day': {
        return currentDate.toLocaleDateString('en-US', {
          weekday: 'long',
          year: 'numeric',
          month: 'long',
          day: 'numeric'
        });
      }
      case 'week': {
        const start = new Date(currentDate);
        start.setDate(currentDate.getDate() - currentDate.getDay());
        const end = new Date(start);
        end.setDate(start.getDate() + 6);
        return `${start.toLocaleDateString()} - ${end.toLocaleDateString()}`;
      }
      case 'month': {
        return currentDate.toLocaleDateString('en-US', { 
          month: 'long', 
          year: 'numeric' 
        });
      }
      default: {
        return currentDate.toLocaleDateString();
      }
    }
  };

  const isPrevDisabled = () => {
    const oneYearAgo = new Date();
    oneYearAgo.setFullYear(oneYearAgo.getFullYear() - 1);
    return currentDate <= oneYearAgo;
  };

  // Data loading
  const loadTodos = async () => {
    try {
      setLoading(true);
      const dateStr = currentDate.toISOString().split('T')[0];
      const response = await fetch(
        `${API_URL}/todos/by-date?range=${dateRange}&date=${dateStr}`
      );
      
      if (!response.ok) throw new Error('Failed to fetch todos');
      
      const data = await response.json();
      
      if (dateRange === 'day') {
        setTodos(data || []);
        setDateGroups([]);
      } else {
        setDateGroups(data || []);
        setTodos([]);
      }
      
      setError(null);
    } catch (err) {
      setError(err.message);
      setTodos([]);
      setDateGroups([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTodos();
  }, [currentDate, dateRange]);

  // Drag and drop handlers
  const handleDragStart = (e, todo) => {
    setDraggedItem(todo);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/html', e.target.outerHTML);
    e.target.style.opacity = '0.5';
  };

  const handleDragEnd = (e) => {
    e.target.style.opacity = '1';
    setDraggedItem(null);
    setDragOverItem(null);
  };

  const handleDragOver = (e) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  };

  const handleDragEnter = (e, todo) => {
    e.preventDefault();
    setDragOverItem(todo);
  };

  const handleDragLeave = (e) => {
    e.preventDefault();
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX;
    const y = e.clientY;
    
    if (x < rect.left || x > rect.right || y < rect.top || y > rect.bottom) {
      setDragOverItem(null);
    }
  };

  const handleDrop = (e, targetTodo) => {
    e.preventDefault();
    
    if (!draggedItem || draggedItem.id === targetTodo.id) {
      return;
    }

    const sourceArray = dateRange === 'day' ? todos : 
                      dateGroups.find(g => g.todos.some(t => t.id === draggedItem.id))?.todos || [];
    
    const targetArray = dateRange === 'day' ? todos : 
                      dateGroups.find(g => g.todos.some(t => t.id === targetTodo.id))?.todos || [];

    const draggedIndex = sourceArray.findIndex(todo => todo.id === draggedItem.id);
    const targetIndex = targetArray.findIndex(todo => todo.id === targetTodo.id);

    if (draggedIndex === -1 || targetIndex === -1) {
      return;
    }

    if (dateRange === 'day') {
      const newTodos = [...todos];
      const [removed] = newTodos.splice(draggedIndex, 1);
      newTodos.splice(targetIndex, 0, removed);
      setTodos(newTodos);
    } else {
      const newDateGroups = [...dateGroups];
      const sourceGroupIndex = newDateGroups.findIndex(g => g.todos.some(t => t.id === draggedItem.id));
      const targetGroupIndex = newDateGroups.findIndex(g => g.todos.some(t => t.id === targetTodo.id));
      
      if (sourceGroupIndex === targetGroupIndex) {
        const newTodos = [...newDateGroups[sourceGroupIndex].todos];
        const [removed] = newTodos.splice(draggedIndex, 1);
        newTodos.splice(targetIndex, 0, removed);
        newDateGroups[sourceGroupIndex].todos = newTodos;
      } else {
        const sourceTodos = [...newDateGroups[sourceGroupIndex].todos];
        const targetTodos = [...newDateGroups[targetGroupIndex].todos];
        const [removed] = sourceTodos.splice(draggedIndex, 1);
        targetTodos.splice(targetIndex, 0, removed);
        newDateGroups[sourceGroupIndex].todos = sourceTodos;
        newDateGroups[targetGroupIndex].todos = targetTodos;
      }
      
      setDateGroups(newDateGroups);
    }

    setDraggedItem(null);
    setDragOverItem(null);
  };

  // CRUD Operations
  const addTodo = async (e) => {
    e.preventDefault();
    if (!title.trim()) return;

    try {
      const newTodo = await createTodo({ title, description, completed: false });
      insertTodoIntoState(newTodo);
      clearForm();
    } catch (err) {
      setError(err.message);
    }
  };

  const createTodo = async (todoData) => {
    const response = await fetch(`${API_URL}/todos`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(todoData),
    });

    if (!response.ok) throw new Error('Failed to create todo');
    return response.json();
  };

  const insertTodoIntoState = (newTodo) => {
    if (dateRange === 'day') {
      setTodos(prev => [newTodo, ...prev]);
    } else {
      const today = getTodayDate();
      const groupIndex = dateGroups.findIndex(g => g.date === today);

      if (groupIndex >= 0) {
        const updatedGroups = [...dateGroups];
        updatedGroups[groupIndex].todos = [newTodo, ...updatedGroups[groupIndex].todos];
        setDateGroups(updatedGroups);
      } else {
        setDateGroups(prev => [{ date: today, todos: [newTodo] }, ...prev]);
      }
    }
  };

  const getTodayDate = () => {
    return new Date().toISOString().split('T')[0];
  };

  const clearForm = () => {
    setTitle('');
    setDescription('');
  };

  const updateTodo = async (updatedTodo) => {
    try {
      const updated = await sendUpdateRequest(updatedTodo);
      updateTodoState(updatedTodo.id, updated);
      setEditingId(null);
    } catch (err) {
      setError(err.message);
    }
  };

  const sendUpdateRequest = async (todo) => {
    const response = await fetch(`${API_URL}/todos/${todo.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(todo)
    });

    if (!response.ok) {
      throw new Error('Failed to update todo');
    }

    return response.json();
  };

  const updateTodoState = (id, updated) => {
    if (dateRange === 'day') {
      setTodos(prev => updateFlatTodos(prev, id, updated));
    } else {
      setDateGroups(prev => updateGroupedTodos(prev, id, updated));
    }
  };

  const updateFlatTodos = (todos, id, updated) => {
    return todos.map(t => (t.id === id ? updated : t));
  };

  const updateGroupedTodos = (groups, id, updated) => {
    return groups.map(group => ({
      ...group,
      todos: updateFlatTodos(group.todos, id, updated)
    }));
  };

  const toggleTodo = (id, completed) => {
    const todo = findTodoById(id);
    if (!todo) return;

    updateTodo({ ...todo, completed });
  };

  const findTodoById = (id) => {
    if (dateRange === 'day') {
      return todos.find(t => t.id === id);
    }

    return dateGroups.flatMap(group => group.todos).find(t => t.id === id);
  };

  const deleteTodo = async (id) => {
    const confirmed = confirmDelete();
    if (!confirmed) return;

    try {
      await deleteTodoFromServer(id);
      removeTodoFromState(id);
    } catch (err) {
      setError('Failed to delete todo');
    }
  };

  const confirmDelete = () => {
    return window.confirm('Are you sure you want to delete this task?');
  };

  const deleteTodoFromServer = async (id) => {
    const response = await fetch(`${API_URL}/todos/${id}`, { method: 'DELETE' });
    if (!response.ok) throw new Error('Delete request failed');
  };

  const removeTodoFromState = (id) => {
    if (dateRange === 'day') {
      setTodos(prev => removeFromFlatList(prev, id));
    } else {
      setDateGroups(prev => removeFromGroupedList(prev, id));
    }
  };

  const removeFromFlatList = (todos, id) => {
    return todos.filter(t => t.id !== id);
  };

  const removeFromGroupedList = (groups, id) => {
    return groups
      .map(group => ({
        ...group,
        todos: removeFromFlatList(group.todos, id)
      }))
      .filter(group => group.todos.length > 0);
  };

  // Rendering functions
  const renderTodoItem = (todo) => (
    <div 
      key={todo.id} 
      className={`todo-item ${todo.completed ? 'completed' : ''} ${
        dragOverItem && dragOverItem.id === todo.id ? 'drag-over' : ''
      }`}
      draggable={editingId !== todo.id}
      onDragStart={(e) => handleDragStart(e, todo)}
      onDragEnd={handleDragEnd}
      onDragOver={handleDragOver}
      onDragEnter={(e) => handleDragEnter(e, todo)}
      onDragLeave={handleDragLeave}
      onDrop={(e) => handleDrop(e, todo)}
    >
      {editingId === todo.id ? (
        <EditTodoForm
          todo={todo}
          onSave={updateTodo}
          onCancel={() => setEditingId(null)}
        />
      ) : (
        <>
          <div className="todo-header">
            <span className="drag-handle" title="Drag to reorder">⋮⋮</span>
            <input
              type="checkbox"
              className="todo-checkbox"
              checked={todo.completed}
              onChange={(e) => toggleTodo(todo.id, e.target.checked)}
            />
            <div className="todo-title">{todo.title}</div>
          </div>
          {todo.description && (
            <div className="todo-description">{todo.description}</div>
          )}
          <div className="todo-actions">
            <button onClick={() => setEditingId(todo.id)} className="edit-btn">
              <span>✏️</span> Edit
            </button>
            <button className="danger" onClick={() => deleteTodo(todo.id)}>
              <span>🗑️</span> Delete
            </button>
          </div>
          <div className="todo-meta">
            <span>Created: {new Date(todo.created_at).toLocaleDateString()}</span>
            <span>Updated: {new Date(todo.updated_at).toLocaleDateString()}</span>
          </div>
        </>
      )}
    </div>
  );

  const renderTodoListContent = () => {
    if (loading) {
      return (
        <div className="loading">
          <div className="spinner"></div>
          Loading your todos...
        </div>
      );
    }

    if (error) {
      return (
        <div className="empty-state">
          <div style={{ fontSize: '4em', marginBottom: '20px' }}>⚠️</div>
          <h3>Error</h3>
          <p>{error}</p>
          <button onClick={loadTodos}>Retry</button>
        </div>
      );
    }

    if (dateGroups.length > 0) {
      return dateGroups.map((group) => (
        <div key={group.date} className="date-group">
          <div className="date-header">
            {new Date(group.date).toLocaleDateString('en-US', {
              weekday: 'long',
              year: 'numeric',
              month: 'long',
              day: 'numeric'
            })}
          </div>
          {group.todos.map(renderTodoItem)}
        </div>
      ));
    }

    if (todos.length === 0) {
      return (
        <div className="empty-state">
          <div style={{ fontSize: '4em', marginBottom: '20px' }}>📝</div>
          <h3>No tasks for this period</h3>
          <p>Add a new task using the form on the right!</p>
        </div>
      );
    }

    return todos.map(renderTodoItem);
  };

  // Stats calculation - Fixed with null safety
  const safeTodos = todos || [];
  const safeDateGroups = dateGroups || [];
  
  const total = safeTodos.length + safeDateGroups.reduce((sum, group) => sum + (group.todos?.length || 0), 0);
  const completed = safeTodos.filter(t => t.completed).length + 
                   safeDateGroups.reduce((sum, group) => sum + (group.todos?.filter(t => t.completed).length || 0), 0);
  const pending = total - completed;

  return (
    <div className="container">
      <div className={`main-content ${sidebarCollapsed ? 'expanded' : ''}`}>
        <h1>MinimalDo</h1>
        
        <div className="date-controls">
          <button 
            onClick={() => navigateDate('prev')}
            disabled={isPrevDisabled()}
            className="nav-btn"
          >
            ← Previous
          </button>
          
          <h2 className="current-period">
            {formatPeriodHeader()}
          </h2>
          
          <button 
            onClick={() => navigateDate('next')}
            className="nav-btn"
          >
            Next →
          </button>
          
          <select 
            value={dateRange} 
            onChange={handleRangeChange}
            className="range-selector"
          >
            <option value="day">Daily</option>
            <option value="week">Weekly</option>
            <option value="month">Monthly</option>
          </select>
        </div>

        <div className="stats">
          <div className="stat-card">
            <div className="stat-number">{total}</div>
            <div className="stat-label">Total Tasks</div>
          </div>
          <div className="stat-card">
            <div className="stat-number">{completed}</div>
            <div className="stat-label">Completed</div>
          </div>
          <div className="stat-card">
            <div className="stat-number">{pending}</div>
            <div className="stat-label">Pending</div>
          </div>
        </div>

        <div id="todoList" className="todo-list">
          {renderTodoListContent()}
        </div>
      </div>

      <div className={`sidebar ${sidebarCollapsed ? 'collapsed' : ''}`}> 
        <h2>✨ Add New Task</h2>
        <form onSubmit={addTodo} className="todo-form">
          <div className="form-group">
            <label htmlFor="title">Task Title *</label>
            <input 
              id="title" 
              type="text" 
              value={title} 
              onChange={(e) => setTitle(e.target.value)} 
              required 
            />
          </div>
          <div className="form-group">
            <label htmlFor="description">Description</label>
            <textarea 
              id="description" 
              value={description} 
              onChange={(e) => setDescription(e.target.value)} 
            />
          </div>
          <button type="submit">
            <span>➕</span> Add Task
          </button>
        </form>
      </div>

      <button className={`toggle-btn ${sidebarCollapsed ? 'sidebar-closed' : 'sidebar-open'}`} onClick={() => setSidebarCollapsed(!sidebarCollapsed)}>
        {sidebarCollapsed ? '➕' : '✕'}
      </button>
    </div>
  );
}

export default App;
