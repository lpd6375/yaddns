async function init() {
  const msgEl = document.getElementById('msg')
  try {
    const savedToken = sessionStorage.getItem('admin_token') || ''
    if (savedToken) document.getElementById('admin_token').value = savedToken

    const res = await fetch('/admin/config', { headers: savedToken ? { 'Authorization': 'Bearer ' + savedToken } : {} })
    if (!res.ok) {
      msgEl.innerText = '无法加载配置';
    } else {
      const cfg = await res.json()

      document.getElementById('interface').value = cfg.Interface || ''
      const cf = cfg.Cloudflare || {}
      document.getElementById('cf_token').value = cf.Token || ''
      document.getElementById('cf_zone_id').value = cf.ZoneID || ''
      document.getElementById('cf_record_id').value = cf.RecordID || ''
      document.getElementById('cf_name').value = cf.Name || ''
      document.getElementById('cf_ttl').value = cf.TTL || ''

      const rt = cfg.Runtime || {}
      document.getElementById('update_interval').value = rt.UpdateInterval || ''
      document.getElementById('admin_addr').value = rt.AdminAddr || ''

      const nt = cfg.Notify || {}
      document.getElementById('wecom_webhook').value = nt.WecomWebhook || ''
      document.getElementById('enable_wecom').checked = !!nt.EnableWecom
      document.getElementById('telegram_token').value = nt.TelegramBotToken || ''
      document.getElementById('telegram_chat_id').value = nt.TelegramChatID || ''
      document.getElementById('enable_telegram').checked = !!nt.EnableTelegram
    }

    document.getElementById('cfgForm').addEventListener('submit', async (e) => {
      e.preventDefault()
      const enteredToken = document.getElementById('admin_token').value || ''
      if (enteredToken) sessionStorage.setItem('admin_token', enteredToken)
      const newCfg = {
        Interface: document.getElementById('interface').value,
        Cloudflare: {
          Token: document.getElementById('cf_token').value,
          ZoneID: document.getElementById('cf_zone_id').value,
          RecordID: document.getElementById('cf_record_id').value,
          Name: document.getElementById('cf_name').value,
          TTL: Number(document.getElementById('cf_ttl').value) || 1
        },
        Runtime: {
          UpdateInterval: Number(document.getElementById('update_interval').value) || 60,
          AdminAddr: document.getElementById('admin_addr').value || ':8080'
        },
        Notify: {
          WecomWebhook: document.getElementById('wecom_webhook').value,
          EnableWecom: document.getElementById('enable_wecom').checked,
          TelegramBotToken: document.getElementById('telegram_token').value,
          TelegramChatID: document.getElementById('telegram_chat_id').value,
          EnableTelegram: document.getElementById('enable_telegram').checked,
        }
      }

      const headers = { 'Content-Type': 'application/json' }
      const authToken = document.getElementById('admin_token').value || ''
      if (authToken) headers['Authorization'] = 'Bearer ' + authToken

      const resp = await fetch('/admin/config', {
        method: 'POST',
        headers: headers,
        body: JSON.stringify(newCfg)
      })
      if (resp.ok) {
        msgEl.innerText = '已保存'
      } else {
        const txt = await resp.text()
        msgEl.innerText = '保存失败: ' + txt
      }
    })

    // backups
    async function loadBackups() {
      const token = sessionStorage.getItem('admin_token') || ''
      const headers = token ? { 'Authorization': 'Bearer ' + token } : {}
      const res = await fetch('/admin/backups', { headers })
      if (!res.ok) {
        document.getElementById('backupsList').innerHTML = '<li>无法获取备份</li>'
        return
      }
      const list = await res.json()
      const ul = document.getElementById('backupsList')
      ul.innerHTML = ''
      list.forEach((n) => {
        const li = document.createElement('li')
        const btn = document.createElement('button')
        btn.innerText = '恢复'
        btn.addEventListener('click', async () => {
          if (!confirm('确认恢复备份: ' + n + ' ?')) return
          const r = await fetch('/admin/backups/restore', {
            method: 'POST',
            headers: Object.assign({ 'Content-Type': 'application/json' }, headers),
            body: JSON.stringify({ file: n })
          })
          if (r.ok) {
            alert('已恢复: ' + n)
            window.location.reload()
          } else {
            const txt = await r.text()
            alert('恢复失败: ' + txt)
          }
        })
        li.innerText = n + ' '
        li.appendChild(btn)
        ul.appendChild(li)
      })
    }

    document.getElementById('refreshBackups').addEventListener('click', loadBackups)
    loadBackups()

    // routing & other sections
    function showSection(name) {
      document.querySelectorAll('.view').forEach((el) => el.style.display = 'none')
      const el = document.getElementById(name)
      if (el) el.style.display = ''
    }

    async function loadHealth() {
      const token = sessionStorage.getItem('admin_token') || ''
      const headers = token ? { 'Authorization': 'Bearer ' + token } : {}
      const res = await fetch('/admin/health', { headers })
      if (!res.ok) return
      const j = await res.json()
      document.getElementById('dashboardContent').innerText = JSON.stringify(j, null, 2)
    }

    async function loadLogs() {
      const token = sessionStorage.getItem('admin_token') || ''
      const headers = token ? { 'Authorization': 'Bearer ' + token } : {}
      const tail = document.getElementById('logsTail').value || '200'
      const res = await fetch('/admin/logs?tail=' + tail, { headers })
      if (!res.ok) {
        document.getElementById('logsContent').innerText = '无法获取日志'
        return
      }
      const lines = await res.json()
      document.getElementById('logsContent').innerText = lines.join('\n')
    }

    async function loadAudit() {
      const token = sessionStorage.getItem('admin_token') || ''
      const headers = token ? { 'Authorization': 'Bearer ' + token } : {}
      const tail = document.getElementById('auditTail').value || '200'
      const res = await fetch('/admin/audit?tail=' + tail, { headers })
      if (!res.ok) {
        document.getElementById('auditContent').innerText = '无法获取审计'
        return
      }
      const arr = await res.json()
      document.getElementById('auditContent').innerText = JSON.stringify(arr, null, 2)
    }

    document.getElementById('refreshLogs').addEventListener('click', loadLogs)
    document.getElementById('refreshAudit').addEventListener('click', loadAudit)

    function route() {
      const hash = (location.hash || '#dashboard').replace('#', '')
      showSection(hash)
      if (hash === 'dashboard') loadHealth()
      if (hash === 'logs') loadLogs()
      if (hash === 'audit') loadAudit()
    }

    window.addEventListener('hashchange', route)
    route()
  } catch (err) {
    msgEl.innerText = '错误: ' + err.message
  }
}

window.addEventListener('load', init)
