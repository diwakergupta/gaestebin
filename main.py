#!/usr/bin/env python

import cgi
import jinja2
import logging
import os
import random
import string
import webapp2

from google.appengine.ext import db
from google.appengine.api import users

jinja_environment = jinja2.Environment(
    loader=jinja2.FileSystemLoader(os.path.dirname(__file__)))

class Paste(db.Model):
    id = db.StringProperty()
    content = db.TextProperty()
    timestamp = date = db.DateTimeProperty(auto_now_add=True)

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
        logging.info("Paste %s, %s", paste.id, paste.content)
        paste.put()
        self.redirect('/' + paste.id)

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
        logging.info("Querying paste_id %s", paste_id)
        query = db.Query(Paste)
        query.filter("id = ", paste_id)
        template_values = {"content": cgi.escape(query.get().content)}
        template = jinja_environment.get_template('index.html')
        self.response.out.write(template.render(template_values))

app = webapp2.WSGIApplication([
    (r'/', CreatePaste),
    (r'/paste', SavePaste),
    (r'/(\S+)', ShowPaste)
    ])
