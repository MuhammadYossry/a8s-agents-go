# 🐍 Python Assistant

```{{ response }}
```

{% if suggestedActions %}
## 🔄 Next Steps
{% for action in suggestedActions %}
- [ ] {{ action }}
{% endfor %}
{% endif %}

---

| Metric | Details |
|--------|---------|
| Confidence | {{ confidence | format }}% |
| Language | {{ language | default 'Python' }} |
| Actions | {{ suggestedActions | length }} |

*🤖 Generated with {{ confidence | format }}% confidence*