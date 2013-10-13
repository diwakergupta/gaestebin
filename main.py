# Copyright 2012 Diwaker Gupta
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import cgi
import jinja2
import os
import random
import string
import webapp2

from google.appengine.api import memcache
from google.appengine.api import users
from google.appengine.ext import db

jinja_environment = jinja2.Environment(
    loader=jinja2.FileSystemLoader(os.path.dirname(__file__)))

class Paste(db.Model):
    id = db.StringProperty()
    content = db.TextProperty()
    timestamp = date = db.DateTimeProperty(auto_now_add=True)
    email = db.EmailProperty()

    def to_memcache(self):
        if self.email:
            # New schema includes attribution metadata
            return {'content': self.content,
                    'email': self.email,
                    'ts': self.timestamp}
        return {'content': self.content}

def gen_random_string(length):
    chars = string.letters + string.digits
    return ''.join(random.choice(chars) for i in xrange(length))

class SavePaste(webapp2.RequestHandler):
    def post(self):
        user = users.get_current_user()
        if not user:
            self.redirect(users.create_login_url(self.request.uri))
        paste = Paste()
        paste.id = gen_random_string(8)
        paste.content = self.request.get('content')
        paste.email = user.email()
        paste.put()
        memcache.add(paste.id, paste.to_memcache())
        self.response.set_cookie('delid', str(paste.key()))
        self.redirect('/' + paste.id)

class DelPaste(webapp2.RequestHandler):
    def post(self):
        user = users.get_current_user()
        if not user:
            self.redirect(users.create_login_url(self.request.uri))
        delid = self.request.get('delid')
        if delid:
            paste = Paste.get(delid)
            if paste:
                memcache.delete(paste.id)
                paste.delete()
        self.redirect('/')

class CreatePaste(webapp2.RequestHandler):
    def get(self):
        user = users.get_current_user()
        if not user:
            self.redirect(users.create_login_url(self.request.uri))
        template_values = {}
        template = jinja_environment.get_template('index.html')
        self.response.out.write(template.render(template_values))

class ShowPaste(webapp2.RequestHandler):
    def get(self, paste_id):
        paste = memcache.get(paste_id)
        if paste is None or not isinstance(paste, dict):
            query = db.Query(Paste)
            query.filter("id = ", paste_id)
            entry = query.get()
            if entry is None:
                self.abort(404)
            paste = entry.to_memcache()
            memcache.set(paste_id, paste)

        # Don't show the email if the user is not authenticated
        user = users.get_current_user()
        template_values = {k: cgi.escape(str(v)) for k, v in paste.iteritems()
                if k != 'email' or user}

        if 'delid' in self.request.cookies:
            template_values['delid'] = self.request.cookies.get('delid')
            self.response.delete_cookie('delid')
        template = jinja_environment.get_template('index.html')
        self.response.out.write(template.render(template_values))

app = webapp2.WSGIApplication([
    (r'/', CreatePaste),
    (r'/paste', SavePaste),
    (r'/oops', DelPaste),
    (r'/(\S+)', ShowPaste)
    ])
