FROM nginx:alpine

# Supprime la page par défaut
RUN rm -rf /usr/share/nginx/html/*

# Copie notre page
COPY index.html /usr/share/nginx/html/index.html

# Expose le port 80
EXPOSE 80
