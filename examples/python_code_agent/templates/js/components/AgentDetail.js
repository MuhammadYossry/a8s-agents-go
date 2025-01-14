// Utility Components
const Badge = ({ children, variant = 'default', className = '' }) => {
  const styles = {
    default: 'bg-blue-100 text-blue-800',
    success: 'bg-green-100 text-green-800',
    purple: 'bg-purple-100 text-purple-800',
    gray: 'bg-gray-100 text-gray-800'
  };
  
  return React.createElement('span', {
    className: `px-2 py-1 rounded-full text-sm font-medium ${styles[variant]} ${className}`
  }, children);
};

const Breadcrumb = ({ items }) => {
  return React.createElement('nav', { className: 'mb-4' },
    React.createElement('ol', { className: 'flex space-x-2 text-gray-600' },
      items.map((item, index) => 
        React.createElement('li', { key: index, className: 'flex items-center' },
          index > 0 && React.createElement('span', { className: 'mx-2' }, '/'),
          item.href 
            ? React.createElement('a', { 
                href: item.href,
                className: 'hover:text-blue-600 transition-colors' 
              }, item.text)
            : React.createElement('span', { 
                className: 'text-gray-900 font-medium' 
              }, item.text)
        )
      )
    )
  );
};

const SkillBadge = ({ path }) => (
  React.createElement('div', { className: 'flex flex-wrap gap-2' },
    path.map((skill, idx) => 
      React.createElement(React.Fragment, { key: skill },
        React.createElement(Badge, { variant: 'purple' }, skill),
        idx < path.length - 1 && 
          React.createElement('span', { className: 'text-purple-400' }, '→')
      )
    )
  )
);

// Loading States
const SkeletonCard = () => (
  React.createElement('div', {
    className: 'bg-white rounded-lg shadow p-6 space-y-4'
  },
    React.createElement('div', { className: 'skeleton h-8 w-48' }),
    React.createElement('div', { className: 'skeleton h-4 w-full' }),
    React.createElement('div', { className: 'skeleton h-24 w-full' })
  )
);

// Main Components
const CapabilityCard = ({ capability }) => {
  if (!capability?.skill_path) return null;

  return React.createElement('div', {
    className: 'bg-white p-6 rounded-lg shadow-sm border card-hover'
  },
    React.createElement(SkillBadge, { path: capability.skill_path }),
    capability.metadata && React.createElement('div', {
      className: 'mt-4 space-y-2'
    },
      Object.entries(capability.metadata).map(([key, values]) => 
        React.createElement('div', { key, className: 'space-y-1' },
          React.createElement('span', {
            className: 'text-sm font-medium text-gray-700'
          }, _.startCase(key)),
          React.createElement('div', {
            className: 'flex flex-wrap gap-2'
          },
            Array.isArray(values) && values.map((value, i) =>
              React.createElement(Badge, {
                key: i,
                variant: 'gray',
                className: 'text-xs'
              }, value)
            )
          )
        )
      )
    )
  );
};

const ActionCard = ({ action, onSelect }) => {
  const getActionIcon = (type) => {
    const icons = {
      talk: 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z',
      search: 'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z',
      generate: 'M13.5 16.5L21 9m0 0l-3-3m3 3h-7.5M10.5 7.5L3 15m0 0l3 3m-3-3h7.5',
      default: 'M9 17.25v1.007a3 3 0 01-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0115 18.257V17.25m6-12V15a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 15V5.25m18 0A2.25 2.25 0 0018.75 3H5.25A2.25 2.25 0 003 5.25m18 0V12a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 12V5.25'
    };
    return icons[type.toLowerCase()] || icons.default;
  };

  return React.createElement('div', {
    className: 'bg-white rounded-lg shadow-sm border p-6 card-hover'
  },
    React.createElement('div', { 
      className: 'flex items-start gap-4'
    },
      React.createElement('div', {
        className: 'p-2 bg-blue-50 rounded-lg'
      },
        React.createElement('svg', {
          className: 'w-6 h-6 text-blue-600',
          fill: 'none',
          viewBox: '0 0 24 24',
          stroke: 'currentColor'
        },
          React.createElement('path', {
            strokeLinecap: 'round',
            strokeLinejoin: 'round',
            strokeWidth: 2,
            d: getActionIcon(action.actionType)
          })
        )
      ),
      React.createElement('div', { className: 'flex-1' },
        React.createElement('div', { 
          className: 'flex items-center justify-between mb-2'
        },
          React.createElement('h3', { 
            className: 'font-semibold text-lg'
          }, action.name),
          React.createElement(Badge, { 
            variant: action.actionType === 'talk' ? 'success' : 'default'
          }, action.actionType)
        ),
        React.createElement('p', {
          className: 'text-gray-600 mb-4'
        }, action.description),
        React.createElement('div', {
          className: 'flex items-center gap-2 text-sm text-gray-500'
        },
          React.createElement(Badge, { variant: 'gray' }, action.method),
          React.createElement('code', {
            className: 'px-2 py-1 bg-gray-100 rounded'
          }, action.path)
        ),
        React.createElement('button', {
          className: 'mt-4 w-full px-4 py-2 text-blue-600 border border-blue-600 rounded-lg hover:bg-blue-50 transition-colors',
          onClick: () => onSelect(action)
        }, 'Try It')
      )
    )
  );
};

// Main Component
function AgentDetail() {
  const [agent, setAgent] = React.useState(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState(null);

  const fetchAgent = async () => {
    try {
      const slug = window.location.pathname.split('/').pop();
      const manifestUrl = `/agents/${slug}/manifest.json`;
      const response = await fetch(manifestUrl);
      if (!response.ok) throw new Error('Failed to fetch agent data');
      const data = await response.json();
      setAgent(data);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  React.useEffect(() => {
    fetchAgent();
  }, []);

  if (error) {
    return React.createElement('div', {
      className: 'min-h-screen flex items-center justify-center'
    },
      React.createElement('div', {
        className: 'bg-red-50 text-red-600 p-4 rounded-lg'
      }, `Error: ${error}`)
    );
  }

  if (loading) {
    return React.createElement('div', {
      className: 'max-w-7xl mx-auto px-4 py-8 sm:px-6 lg:px-8'
    },
      React.createElement('div', {
        className: 'space-y-6'
      },
        React.createElement(SkeletonCard),
        React.createElement('div', {
          className: 'grid grid-cols-1 lg:grid-cols-2 gap-6'
        },
          Array(4).fill(null).map((_, i) => 
            React.createElement(SkeletonCard, { key: i }))
        )
      )
    );
  }

  return React.createElement('div', {
    className: 'min-h-screen bg-gray-50 py-8'
  },
    React.createElement('div', {
      className: 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8'
    },
      React.createElement(Breadcrumb, {
        items: [
          { text: 'Home', href: '/' },
          { text: 'Agents', href: '/agents' },
          { text: agent?.name }
        ]
      }),
      React.createElement('div', {
        className: 'space-y-8'
      },
        // Header Section
        React.createElement('header', {
          className: 'bg-white rounded-lg shadow-sm border p-6'
        },
          React.createElement('div', {
            className: 'flex items-start gap-6'
          },
            React.createElement('div', {
              className: 'flex-1'
            },
              React.createElement('div', {
                className: 'flex items-center gap-3 mb-2'
              },
                React.createElement('h1', {
                  className: 'text-2xl font-bold'
                }, agent?.name),
                React.createElement(Badge, {
                  variant: 'purple'
                }, `v${agent?.version || '1.0.0'}`)
              ),
              React.createElement('p', {
                className: 'text-gray-600'
              }, agent?.description)
            )
          )
        ),
        // Capabilities Section
        React.createElement('section', null,
          React.createElement('h2', {
            className: 'text-xl font-bold mb-4'
          }, 'Capabilities'),
          React.createElement('div', {
            className: 'grid grid-cols-1 lg:grid-cols-2 gap-6'
          },
            agent?.capabilities?.map((capability, idx) =>
              React.createElement(CapabilityCard, {
                key: idx,
                capability: capability
              })
            )
          )
        ),
        // Actions Section
        React.createElement('section', null,
          React.createElement('h2', {
            className: 'text-xl font-bold mb-4'
          }, 'Available Actions'),
          React.createElement('div', {
            className: 'grid grid-cols-1 lg:grid-cols-2 gap-6'
          },
            agent?.actions?.map((action, idx) =>
              React.createElement(ActionCard, {
                key: idx,
                action: action,
                onSelect: () => window.location.href = `/agents/${agent.slug}/action/${action.slug}`
              })
            )
          )
        )
      )
    )
  );
}