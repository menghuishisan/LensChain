FROM php:8.3-apache
WORKDIR /var/www/html
RUN docker-php-ext-install mysqli pdo pdo_mysql
COPY index.php /var/www/html/index.php
EXPOSE 8080
RUN sed -ri 's/Listen 80/Listen 8080/' /etc/apache2/ports.conf /etc/apache2/sites-available/000-default.conf
CMD ["apache2-foreground"]
